package future_test

import (
	"testing"

	task "github.com/Ar37-rs/future"
)

func TestDrive(t *testing.T) {
	test_task := task.Future(7, func(it *task.ITask) task.Progress {
		for i := 0; i < 5; i++ {
			it.Send(i)
		}
		return task.Completed("Ok")
	})

	if test_task.Id() != 7 {
		t.Fatalf(`"task id test failure! with value: %d"`, test_task.Id())
	}

	test_task.TryDo()
	quit := false
	count := 0

	for {
		test_task.TryResolve(func(p *task.Progress, done bool) {

			p.Current(func(v task.Value) {
				k := v.(int)
				count += k
			})

			p.OnCompleted(func(v task.Value) {
				// Check if return OnCompleted failure
				if v.(string) != "Ok" {
					t.Fatalf(`"OnCompleted test failure! with value: %s"`, v.(string))
				}
			})

			if done {
				// Check if counter races
				if count != 10 {
					t.Fatalf(`"Sender (channel) test failure! count value: %d"`, count)
				}
				quit = true
			}
		})

		if quit {
			break
		}
	}
}
