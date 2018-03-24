package ctrl

import (
	"sync"
	"log"
)

var goroutines_wait *sync.WaitGroup

func init(){
	goroutines_wait = new(sync.WaitGroup)
}

func GlobalWaitGroup()*sync.WaitGroup{
	if goroutines_wait!=nil{
		return goroutines_wait
	}else{
		log.Fatalln("Wait group is <nil>.")
		return nil
	}
}

type Closure func()

func StartGoroutines(f Closure){
	go func() {
		GlobalWaitGroup().Add(1)
		f()
		goroutines_wait.Done()
	}()
}


