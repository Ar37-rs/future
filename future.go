// github.com/Ar37-rs/future

package future

import (
	"sync"
	"sync/atomic"
	"time"
)

// Future value
type Value = interface{}

// Inner Task
type ITask struct {
	id                          uint
	suspend, cancel, sync_ready *atomic.Value
	current_progress            Value
	sync_wg                     sync.WaitGroup
	receiver                    *channel
}

// Get id of the task
func (it *ITask) Id() uint {
	return it.id
}

// Check if progress of the task should be canceled.
func (it *ITask) ShouldCancel() bool {
	return it.cancel.Load().(bool)
}

// Check if the task should be suspended,
//
// usually applied for a specific task with event loop in it.
//
// do other things (switch) while the task is suspended or Wait() the task immediately.
func (it *ITask) ShouldSuspend() bool {
	return it.suspend.Load().(bool)
}

// Wait immediately if the task should be suspended and will do nothing if it should not.
func (it *ITask) Wait() {
	if it.suspend.Load().(bool) {
		it.sync_wg.Wait()
	}
}

// Send current inner task progress
func (it *ITask) Send(value interface{}) {
	it.sync_wg.Add(1)
	it.current_progress = value
	it.sync_ready.Store(true)
	it.sync_wg.Wait()
}

// Receive value from the outer task (if any, and will return nil if nothing).
func (it *ITask) Recv() Value {
	// No need mutex, it's only bi directional data passing
	val := it.receiver.data
	it.receiver.data = nil
	return val
}

type channel struct {
	data Value
}

// Future task
type Task struct {
	awaiting, ready, changed *atomic.Value
	inner_task               *ITask
	fn                       func(*ITask) Progress
	progress                 *Progress
}

// Create new future task
func Future(id uint, fn func(f *ITask) Progress) *Task {
	awaiting := &atomic.Value{}
	ready := &atomic.Value{}
	cancel := &atomic.Value{}
	suspend := &atomic.Value{}
	sync_ready := &atomic.Value{}
	changed := &atomic.Value{}
	_channel := &channel{data: nil}
	ready.Store(false)
	awaiting.Store(false)
	cancel.Store(false)
	suspend.Store(false)
	sync_ready.Store(false)
	changed.Store(false)
	var sync_wg sync.WaitGroup
	inner_task := &ITask{id: id, suspend: suspend, cancel: cancel, current_progress: nil, sync_wg: sync_wg, sync_ready: sync_ready, receiver: _channel}
	return &Task{awaiting: awaiting, ready: ready, changed: changed, inner_task: inner_task, fn: fn, progress: &Progress{current: nil}}
}

func run(f *Task) {
	*f.progress = f.fn(f.inner_task)
	f.ready.Store(true)
}

// Try do the task now (it won't block the current thread)
//
// and then TryResolve later.
func (f *Task) TryDo() {
	if !f.awaiting.Load().(bool) {
		f.awaiting.Store(true)
		f.inner_task.cancel.Store(false)
		go run(f)
	}
}

// Send value to the inner task
func (it *Task) Send(value interface{}) {
	it.inner_task.receiver.data = value
}

// Change task, make sure the task isn't in progress.
func (f *Task) ChangeTask(fn func(f *ITask) Progress) {
	if !f.awaiting.Load().(bool) {
		*f.progress = Progress{current: nil, cancel: nil, complete: nil, _error: nil}
		f.inner_task.current_progress = nil
		f.ready.Store(false)
		f.inner_task.cancel.Store(false)
		f.inner_task.suspend.Store(false)
		f.inner_task.sync_ready.Store(false)
		f.fn = fn
		f.changed.Store(true)
	}
}

// Check if the task is in progress
func (f *Task) IsInProgress() bool {
	return f.awaiting.Load().(bool)
}

// Check if the task is changed
func (f *Task) IsChanged() bool {
	if f.changed.Load().(bool) {
		f.changed.Store(false)
		return true
	} else {
		return false
	}
}

// Check if the task is done (not in progress anymore)
func (f *Task) IsDone() bool {
	return !f.awaiting.Load().(bool)
}

// Try resolve progress of the future task (non-blocking)
func (f *Task) TryResolve(fn func(*Progress, bool)) {
	if f.awaiting.Load().(bool) {
		if f.ready.Load().(bool) {
			f.ready.Store(false)
			f.inner_task.suspend.Store(false)
			fn(f.progress, true)
			f.awaiting.Store(false)
			*f.progress = Progress{current: nil, cancel: nil, complete: nil, _error: nil}
		} else if f.inner_task.sync_ready.Load().(bool) {
			f.progress.current = f.inner_task.current_progress
			fn(f.progress, false)
			f.inner_task.current_progress = nil
			f.progress.current = nil
			f.inner_task.sync_ready.Store(false)
			f.inner_task.sync_wg.Done()
		} else {
			fn(f.progress, false)
		}
	}
}

// Careful! this is blocking operation, resolve progress of the future task (will "block" until the task is completed).
//
// if spin_delay < 3 milliseconds, spin_delay value will be 3 milliseconds, set it large than that or accordingly.
//
// due to blocking operation nature this fn only return whether the task is canceled, completed or error. current progress (sender) will be ignored.
func (f *Task) WaitResolve(spin_delay uint, fn func(*Progress, bool)) {
	if f.awaiting.Load().(bool) {
		del := spin_delay
		if spin_delay < 3 {
			del = 3
		}
		_spin_delay := time.Millisecond * time.Duration(del)
		for {
			if f.ready.Load().(bool) {
				f.ready.Store(false)
				f.inner_task.suspend.Store(false)
				fn(f.progress, true)
				f.awaiting.Store(false)
				*f.progress = Progress{current: nil, cancel: nil, complete: nil, _error: nil}
				f.inner_task.current_progress = nil
			} else if f.IsDone() {
				break
			}
			if f.inner_task.sync_ready.Load().(bool) {
				f.inner_task.sync_ready.Store(false)
				f.inner_task.sync_wg.Done()
			}
			time.Sleep(_spin_delay)
		}
	}
}

// Send signal to the inner task that the task should be suspended.
//
// this won't do anything if not explicitly configured inside the task.
func (f *Task) Suspend() {
	if f.IsInProgress() && !f.inner_task.suspend.Load().(bool) {
		f.inner_task.sync_wg.Add(1)
		f.inner_task.suspend.Store(true)
	}
}

// Resume the suspended task.
//
// this won't do anything if not explicitly configured inside the task.
func (f *Task) Resume() {
	if f.inner_task.suspend.Load().(bool) && f.IsInProgress() {
		f.inner_task.suspend.Store(false)
		f.inner_task.sync_wg.Done()
	}
}

// Check if the task is suspended
func (f *Task) IsSuspended() bool {
	return f.inner_task.suspend.Load().(bool)
}

// Send signal to the inner task that the task should be canceled.
//
// this won't do anything if not explicitly configured inside the task.
func (f *Task) Cancel() {
	f.inner_task.cancel.Store(true)
}

// Check if the task is canceled
func (f *Task) IsCanceled() bool {
	return f.inner_task.cancel.Load().(bool)
}

// Get id of the task
func (f *Task) Id() uint {
	return f.inner_task.id
}

// Progress of the task
type Progress struct {
	current, cancel, complete Value
	_error                    error
}

func Complete(val Value) Progress {
	return Progress{complete: val}
}

func Cancel(val Value) Progress {
	return Progress{cancel: val}
}

func Error(e error) Progress {
	return Progress{_error: e}
}

// Current progress of the task with sender Value (if any)
func (p *Progress) Current(fn func(Value)) {
	if cur := p.current; cur != nil {
		fn(cur)

	}
}

// Indicates the task is completed
func (p *Progress) OnComplete(fn func(Value)) {
	if com := p.complete; com != nil {
		fn(com)
	}
}

// Indicates the task is canceled
func (p *Progress) OnCancel(fn func(Value)) {
	if can := p.cancel; can != nil {
		fn(can)
	}
}

// Indicates the task is error
func (p *Progress) OnError(fn func(error)) {
	if err := p._error; err != nil {
		fn(err)
	}
}

// Inner Runnable Task
type IRTask struct {
	id              uint
	suspend, cancel *atomic.Value
	sync_wg         sync.WaitGroup
}

// Get id of the task
func (it *IRTask) Id() uint {
	return it.id
}

// Check if the runnable task should be canceled.
func (it *IRTask) ShouldCancel() bool {
	return it.cancel.Load().(bool)
}

// Check if the runnable task should be suspended,
//
// usually applied for a specific task with event loop in it.
//
// do other things (switch) while the task is suspended or Wait() the task immediately.
func (it *IRTask) ShouldSuspend() bool {
	return it.suspend.Load().(bool)
}

// Wait immediately if the runnable task should be suspended and will do nothing if it should not.
func (it *IRTask) Wait() {
	if it.suspend.Load().(bool) {
		it.sync_wg.Wait()
	}
}

// Runnable task
type RTask struct {
	running, changed *atomic.Value
	inner_task       *IRTask
	fn               func(*IRTask)
}

// Create new runnable task
func Runnable(id uint, fn func(f *IRTask)) *RTask {
	running := &atomic.Value{}
	suspend := &atomic.Value{}
	cancel := &atomic.Value{}
	changed := &atomic.Value{}
	running.Store(false)
	suspend.Store(false)
	cancel.Store(false)
	changed.Store(false)
	var sync_wg sync.WaitGroup
	inner_task := &IRTask{id: id, suspend: suspend, cancel: cancel, sync_wg: sync_wg}
	return &RTask{running: running, changed: changed, inner_task: inner_task, fn: fn}
}

func _run(f *RTask) {
	f.fn(f.inner_task)
	f.running.Store(false)
	f.inner_task.suspend.Store(false)
}

// Try do the runnable task now (it won't block the current thread)
func (f *RTask) TryDo() {
	if !f.running.Load().(bool) {
		f.running.Store(true)
		f.inner_task.cancel.Store(false)
		go _run(f)
	}
}

// Change the runnable task, make sure the task isn't running.
func (f *RTask) ChangeTask(fn func(f *IRTask)) {
	if !f.running.Load().(bool) {
		f.inner_task.cancel.Store(false)
		f.inner_task.suspend.Store(false)
		f.fn = fn
		f.changed.Store(true)
	}
}

// Check if the task is running
func (f *RTask) IsRunning() bool {
	return f.running.Load().(bool)
}

// Check if the task is changed
func (f *RTask) IsChanged() bool {
	if f.changed.Load().(bool) {
		f.changed.Store(false)
		return true
	} else {
		return false
	}
}

// Check if the runnable task is not running anymore (or done)
func (f *RTask) IsNotRunning() bool {
	return !f.running.Load().(bool)
}

// Send signal to the inner runnable task handle that the task should be suspended.
//
// this won't do anything if not explicitly configured inside the task.
func (f *RTask) Suspend() {
	if f.IsRunning() && !f.inner_task.suspend.Load().(bool) {
		f.inner_task.sync_wg.Add(1)
		f.inner_task.suspend.Store(true)
	}
}

// Resume the suspended runnable task.
//
// this won't do anything if not explicitly configured inside the task.
func (f *RTask) Resume() {
	if f.inner_task.suspend.Load().(bool) && f.IsRunning() {
		f.inner_task.suspend.Store(false)
		f.inner_task.sync_wg.Done()
	}
}

// Check if the task is suspended
func (f *RTask) IsSuspended() bool {
	return f.inner_task.suspend.Load().(bool)
}

// Send signal to the  inner runnable handle that the task should be canceled.
//
// this won't do anything if not explicitly configured inside the task.
func (f *RTask) Cancel() {
	f.inner_task.cancel.Store(true)
}

// Check if the runnable task is canceled
func (f *RTask) IsCanceled() bool {
	return f.inner_task.cancel.Load().(bool)
}

// Get id of the runnable task
func (f *RTask) Id() uint {
	return f.inner_task.id
}
