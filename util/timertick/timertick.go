package timertick

import (
	"wwt/util/heap"
	"wwt/ctrl"
	"time"
	"fmt"
)

const (
	TIMERFUNC_COUNT = 8
)

type TickFunc func()

type TaskPackage struct {
	F         TickFunc
	TimeStamp int64
}

func (this *TaskPackage) Compare(other heap.Compareable) bool {
	t := other.(*TaskPackage)
	return this.TimeStamp < t.TimeStamp
}

var ch chan struct{}

var taskList *heap.SafeHeap

func StartTimerTick() {
	ch = make(chan struct{})
	taskList = heap.NewSafeHeap()

	for i := 0; i < TIMERFUNC_COUNT; i++ {
		ctrl.StartGoroutines(func() {
			timerFunc()
		})
	}
}

func timerFunc() {
	defer func() {
		fmt.Println("Timer Func Exit.")
	}()
	for {
		time.Sleep(time.Second)
		select {
		case <-ch:
			return
		default:
			var task *TaskPackage = nil
			taskList.Lock()
			//	取任务
			if taskList.Len() > 0 {
				task = taskList.Pop().(*TaskPackage)
			}
			taskList.UnLock()

			if task != nil {
				cur := time.Now().UnixNano() / 1e6
				if task.TimeStamp <= cur {
					task.F()
				} else {
					taskList.Lock()
					taskList.AddNode(task)
					taskList.UnLock()
				}
			}
		}
	}
}


func AddTask(tickTime int64, tickFunc TickFunc) {
	task := new(TaskPackage)
	task.F = tickFunc
	task.TimeStamp = tickTime

	taskList.Lock()
	taskList.AddNode(task)
	taskList.UnLock()
}

func CloseTimerTick() {
	close(ch)
	taskList.Lock()
	for taskList.Len() > 0 {
		taskList.Pop()
	}
	taskList.UnLock()
}
