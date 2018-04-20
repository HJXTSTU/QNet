package util

import "sync"

type QMap struct {
	mu sync.RWMutex
	mp map[interface{}]interface{}
}
type RangeFunc func(key interface{},value interface{})
func (this *QMap)Range(f RangeFunc){
	this.mu.Lock()
	defer this.mu.Unlock()
	if len(this.mp)==0{
		return
	}
	for k,v := range this.mp{
		f(k,v)
	}
}

func (this *QMap)Clear(){
	this.mu.Lock()
	defer this.mu.Unlock()
	if len(this.mp)==0{
		return
	}
	for k,_ := range this.mp{
		delete(this.mp, k)
	}
}

func (this *QMap)GetAnyValue()interface{}{
	this.mu.RLock()
	defer this.mu.RUnlock()
	for _,v := range this.mp{
		return v
	}
	return nil
}

func (this *QMap)GetAnyKey()interface{}{
	this.mu.RLock()
	defer this.mu.RUnlock()
	for k,_ := range this.mp{
		return k
	}
	return nil
}

func (this *QMap) Get(k interface{}) interface{} {
	this.mu.RLock()
	defer this.mu.RUnlock()
	if val, ok := this.mp[k]; ok {
		return val
	} else {
		return nil
	}
}

func (this *QMap) Delete(k interface{}) {
	this.mu.Lock()
	defer this.mu.Unlock()
	delete(this.mp, k)
}

func (this *QMap) Set(k, v interface{}) bool {
	this.mu.Lock()
	defer this.mu.Unlock()
	val, ok := this.mp[k]
	if !ok || val != v {
		this.mp[k] = v
	} else {
		return false
	}
	return true
}

func (this *QMap)Length()int{
	this.mu.RLock()
	defer this.mu.RUnlock()
	return len(this.mp)
}

func (this *QMap) HasKey(k interface{}) bool {
	this.mu.RLock()
	defer this.mu.RUnlock()
	_, ok := this.mp[k]
	return ok
}

func NewQMap() *QMap {
	return &QMap{
		sync.RWMutex{},
		make(map[interface{}]interface{}),
	}
}
