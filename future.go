// github.com/Ar37-rs

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
	id                   uint
	suspend, cancel, swr *atomic.Value
	p                    *Progress
	sw                   sync.WaitGroup
}

// Get id of the task
func (it *ITask) Id() uint {
	return it.id
}

// Check if progress of the task should be canceled.
func (it *ITask) ShouldCancel() bool {
	return it.cancel.Load().(bool)
}

// Check if progress of the task should be suspended,
//
// usually applied for a specific task with event loop in it.
//
// do other things (switch) while the task is suspended.
func (it *ITask) ShouldSuspend() bool {
	return it.cancel.Load().(bool)
}

// Send current task progress
func (it *ITask) Send(value interface{}) {
	it.sw.Add(1)
	it.p.current = value
	it.swr.Store(true)
	it.sw.Wait()
}

// Future task
type Task struct {
	id              uint
	awaiting, ready *atomic.Value
	it              *ITask
	fn              func(*ITask) Progress
	p               *Progress
}

// Create new future task
func Future(id uint, fn func(f *ITask) Progress) *Task {
	awaiting := &atomic.Value{}
	ready := &atomic.Value{}
	cancel := &atomic.Value{}
	suspend := &atomic.Value{}
	swr := &atomic.Value{}
	ready.Store(false)
	awaiting.Store(false)
	cancel.Store(false)
	swr.Store(false)
	var sw sync.WaitGroup
	it := &ITask{id: id, suspend: suspend, cancel: cancel, p: &Progress{current: nil}, sw: sw, swr: swr}
	return &Task{id: id, awaiting: awaiting, ready: ready, it: it, fn: fn, p: &Progress{current: nil}}
}

func run(f *Task) {
	*f.p = f.fn(f.it)
	f.ready.Store(true)
}

// Try do the task now (it won't block the current thread)
//
// and then TryResolve later.
func (f *Task) TryDo() {
	if !f.awaiting.Load().(bool) {
		f.awaiting.Store(true)
		f.it.cancel.Store(false)
		go run(f)
	}
}

// Check if the task is in progress
func (f *Task) IsInProgress() bool {
	return f.awaiting.Load().(bool)
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
			f.it.suspend.Store(false)
			fn(f.p, true)
			f.awaiting.Store(false)
			*f.p = Progress{current: nil, cancel: nil, complete: nil, _error: nil}
		} else if f.it.swr.Load().(bool) {
			fn(f.it.p, false)
			*f.it.p = Progress{current: nil, cancel: nil, complete: nil, _error: nil}
			f.it.swr.Store(false)
			f.it.sw.Done()
		} else {
			fn(f.p, false)
		}
	}
}

// Careful! this is blocking operation, resolve progress of the future task (will "block" until the task is completed).
//
// if spin_delay < 3 milliseconds, spin_delay value will be 3 milliseconds, set it large than that or accordingly.
func (f *Task) Resolve(spin_delay uint, fn func(*Progress, bool)) {
	if f.awaiting.Load().(bool) {
		del := spin_delay
		if spin_delay < 3 {
			del = 3
		}
		_spin_delay := time.Millisecond * time.Duration(del)
		for {
			if f.ready.Load().(bool) {
				f.ready.Store(false)
				f.it.suspend.Store(false)
				fn(f.p, true)
				f.awaiting.Store(false)
				*f.p = Progress{current: nil, cancel: nil, complete: nil, _error: nil}
				f.it.p.current = Progress{current: nil, cancel: nil, complete: nil, _error: nil}
			} else if f.IsDone() {
				break
			}
			if f.it.swr.Load().(bool) {
				f.it.sw.Done()
				f.it.swr.Store(false)
			}
			time.Sleep(_spin_delay)
		}
	}
}

// Send signal to the inner task handle that the task should be suspended.
//
// this won't do anything if not explicitly configured inside the task.
func (f *Task) Suspend() {
	f.it.suspend.Store(true)
}

// Resume the suspended task.
//
// this won't do anything if not explicitly configured inside the task.
func (f *Task) Resume() {
	f.it.suspend.Store(false)
}

// Check if the task is suspended
func (f *Task) IsSuspended() {
	f.it.suspend.Store(true)
}

// Send signal to the inner task handle that the task should be canceled.
//
// this won't do anything if not explicitly configured inside the task.
func (f *Task) Cancel() {
	f.it.cancel.Store(true)
}

// Check if the task is canceled
func (f *Task) IsCanceled() bool {
	return f.it.cancel.Load().(bool)
}

// Get id of the task
func (f *Task) Id() uint {
	return f.id
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
