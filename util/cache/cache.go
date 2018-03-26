package cache

import "wwt/util/queue"

type CacheHandler interface {
	Empty() bool
	Size()int
	Clear()

	Add(value interface{})
	Get()interface{}
	Wait()
}

type QCache struct {
	cache queue.QueueHandler
	ch    chan struct{}
}

func (this *QCache) Empty() bool {
	return this.cache.Size() == 0
}

func (this *QCache)Size()int{
	return this.cache.Size()
}

func (this *QCache)Clear(){
	this.cache.Clear()
}

func (this *QCache)Wait(){
	<-this.ch
}

func (this *QCache)Add(value interface{}){
	this.cache.Enqueue(value)
	this.ch<- struct{}{}
}

func (this *QCache)Get()interface{}{
	return this.cache.Dequeue()
}

func New()CacheHandler{
	cache := QCache{queue.New(),make(chan struct{})}
	return &cache
}
