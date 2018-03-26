package queue

import (
	"container/list"
	"sync"
)

type QueueHandler interface {
	Front() interface{}
	Enqueue(value interface{})
	Dequeue() interface{}
	Clear()
	Size() int
}

type Queue struct {
	list *list.List
	mu   sync.Mutex
	size int
}

func (this *Queue) Front() interface{} {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.list.Front().Value
}

func (this *Queue) Enqueue(value interface{}) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.list.PushBack(value)
	this.size++;
}

func (this *Queue) Dequeue() interface{} {
	this.mu.Lock()
	defer this.mu.Unlock()
	front := this.list.Front()
	value := front.Value
	this.list.Remove(front)
	this.size--;
	return value;
}

func (this *Queue) Size() int {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.size
}

func (this *Queue) Clear() {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.list.Init()
	this.size = 0;
}

func New() QueueHandler {
	que := Queue{}
	que.size = 0
	que.list = list.New()
	que.mu = sync.Mutex{}
	return &que
}
