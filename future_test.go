package future_test

import (
	"testing"

	task "github.com/Ar37-rs/future"
)

func TestDrive(t *testing.T) {
	// test future task
	future_task := task.Future(7, func(it *task.ITask) task.Progress {
		for i := 1; i < 5; i++ {
			it.Send(i)
		}
		return task.Complete("Ok")
	})
	if future_task.Id() != 7 {
		t.Fatalf(`"future_task id test failure! with value: %d"`, future_task.Id())
	}
	// try do the task now
	future_task.TryDo()

	// test runnable task
	run_counter := 0
	runnable_task := task.Runnable(8, func(it *task.IRTask) {
		for i := 1; i < 6; i++ {
			run_counter += i
		}
	})
	if runnable_task.Id() != 8 {
		t.Fatalf(`"runnable_task id test failure! with value: %d"`, runnable_task.Id())
	}
	// try do, join the future_task.
	runnable_task.TryDo()

	quit := false
	counter := 0

	for {
		future_task.TryResolve(func(p *task.Progress, done bool) {

			p.Current(func(v task.Value) {
				k := v.(int)
				counter += k
			})

			p.OnComplete(func(v task.Value) {
				// Check if future_task return OnComplete failure
				if v.(string) != "Ok" {
					t.Fatalf(`"future_task OnComplete test failure! with value: %s"`, v.(string))
				}
			})

			if done {
				// Check if counter races
				if counter != 10 {
					t.Fatalf(`"Sender (channel) test failure! count value: %d"`, counter)
				}
				quit = true
			}
		})

		if runnable_task.IsNotRunning() {
			if run_counter != 15 {
				t.Fatalf(`"runnable_task test failure! run_count value: %d"`, run_counter)
			}
		}

		if quit && runnable_task.IsNotRunning() {
			break
		}
	}
}
