# Future

Port of [asynchron](https://github.com/Ar37-rs/asynchron) to go.

# Installation

```sh
go get -u github.com/Ar37-rs/future
```

# Example 

```go
package main

import (
	"fmt"
	"time"
	task "github.com/Ar37-rs/future"
)

func main() {
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
