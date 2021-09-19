# Future

Port of [asynchron](https://github.com/Ar37-rs/asynchron) to go.

# Installation

```sh
go get -u github.com/Ar37-rs/future
```

# Future Task Example 

```go
package main

import (
	"fmt"
	"time"
	task "github.com/Ar37-rs/future"
)

func main() {
	// Create new future task.
	task1 := task.Future(1, func(it *task.ITask) task.Progress {
		count := 0
		for {
			// // return task.Cancel(..) if canceled.
			// if it.ShouldCancel() {
			// 	cancel := fmt.Sprintf(`the task with id: %d canceled`, it.Id())
			// 	return task.Cancel(cancel)
			// }
			count += 1
			time.Sleep(time.Millisecond * 10)
			// Send some messages from the ITask (Inner Task)
			it.Send(fmt.Sprintf(`hello: %d`, count))
			if count > 5 {
				break
			}
		}
		complete := fmt.Sprintf(`the task with id: %d completed`, it.Id())
		return task.Complete(complete)
		// // return error if need to.
		// _error := fmt.Errorf(`the task with id: %d error`, it.Id())
		// return task.Error(_error)
	})
	println(fmt.Sprintf(`task id: %d`, task1.Id()))
	// Try do the task now.
	task1.TryDo()
	// // Cancel if need to.
	// task1.Cancel()
	quit := false
	for {
		println("this event loop won't be blocked.")
		time.Sleep(time.Millisecond * 100)
		task1.TryResolve(func(p *task.Progress, done bool) {
			p.Current(func(v task.Value) {
				println(v.(string))
			})

			p.OnCancel(func(v task.Value) {
				println(v.(string))
			})

			p.OnComplete(func(v task.Value) {
				println(v.(string))
			})

			p.OnError(func(e error) {
				println(e.Error())
			})

			if done {
				quit = true
			}
		})

		if quit {
			break
		}
	}
}
```


# Runnable Task Example 

```go
package main

import (
	"fmt"
	"time"
	task "github.com/Ar37-rs/future"
)

func main() {
	counter := 0
	// Create new runnable task.
	runnable_task := task.Runnable(8, func(it *task.IRTask) {
		for {
			// Make sure the loop not spin too fast, sleep a few millis if need to be canceled.
			time.Sleep(time.Millisecond * 10)
			if it.ShouldCancel() || counter > 5 {
				break
			}
			counter += 1
		}
	})
	if runnable_task.Id() != 8 {
		t.Fatalf(`"runnable_task id test failure! with value: %d"`, runnable_task.Id())
	}
	// Try do the task now.
	runnable_task.TryDo()
	println(fmt.Sprintf(`task id: %d`, runnable_task.Id()))
	// // Cancel if need to.
	// runnable_task.Cancel()
	for {
		if runnable_task.IsNotRunning() {
			break
		}
	}
	if runnable_task.IsCanceled() {
		println("runnable task canceled.")
	}
	println(fmt.Sprintf(`counter value: %d`, counter))
}
```
