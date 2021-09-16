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
	cancel  *atomic.Value
	p       *Progress
	s       sync.WaitGroup
	s_ready *atomic.Value
}

// Check if the task if canceled
func (it *ITask) IsCanceled() bool {
	return it.cancel.Load().(bool)
}

// Send current task progress
func (it *ITask) Send(value interface{}) {
	it.s.Add(1)
	it.p.current = value
	it.s_ready.Store(true)
	it.s.Wait()
}

// Future task
type Task struct {
	id       uint
	awaiting *atomic.Value
	ready    *atomic.Value
	it       *ITask
	fn       func(*ITask) Progress
	p        Progress
}

// Create new future task
func Future(id uint, fn func(f *ITask) Progress) *Task {
	awaiting := &atomic.Value{}
	ready := &atomic.Value{}
	cancel := &atomic.Value{}
	s_ready := &atomic.Value{}
	ready.Store(false)
	awaiting.Store(false)
	cancel.Store(false)
	s_ready.Store(false)
	var s sync.WaitGroup
	it := &ITask{cancel: cancel, p: &Progress{current: nil}, s: s, s_ready: s_ready}
	return &Task{id: id, awaiting: awaiting, ready: ready, it: it, fn: fn, p: Progress{current: nil}}
}

func run(f *Task) {
	f.p = f.fn(f.it)
	f.ready.Store(true)
}

// Try do the task now
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
			f.awaiting.Store(false)
			f.ready.Store(false)
			fn(&f.p, true)
			f.p = Progress{cancel: nil, complete: nil, _error: nil}
		} else if f.it.s_ready.Load().(bool) {
			f.p = *f.it.p
			fn(&f.p, false)
			f.p.current = nil
			f.it.p.current = nil
			f.it.s_ready.Store(false)
			f.it.s.Done()
		} else {
			fn(&f.p, false)
		}
	}
}

// Careful! this is blocking operation, resolve progress of the future task (will "block" until the task is completed).
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
				f.awaiting.Store(false)
				f.ready.Store(false)
				fn(&f.p, true)
				f.p = Progress{cancel: nil, complete: nil, _error: nil}
			} else if f.IsDone() {
				break
			}
			time.Sleep(_spin_delay)
		}
	}
}

// Cancel current task
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
	current  Value
	cancel   Value
	complete Value
	_error   error
}

func Completed(val Value) Progress {
	return Progress{complete: val}
}

func Canceled(val Value) Progress {
	return Progress{cancel: val}
}

func Error(e error) Progress {
	return Progress{_error: e}
}

// Current progress of the task
func (p *Progress) Current(fn func(Value)) {
	cur := p.current
	if cur != nil {
		fn(cur)

	}
}

// Indicates the task is completed
func (p *Progress) OnCompleted(fn func(Value)) {
	com := p.complete
	if com != nil {
		fn(com)
	}
}

// Indicates the task is canceled
func (p *Progress) OnCanceled(fn func(Value)) {
	can := p.cancel
	if can != nil {
		fn(can)
	}
}

// Indicates the task is error
func (p *Progress) OnError(fn func(error)) {
	err := p._error
	if err != nil {
		fn(err)
	}
}
