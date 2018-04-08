package connpool

import (
	"wwt/net/client"
	"wwt/net/server/connection"
	"sync"
)

// 连接池
type ConnectionPoolHandler interface {
	//	对同一个目标终端建立连接
	Connect(address string, count int)

	//	获取一个连接
	GetConnection(token connection.TokenHandler) client.ClientHandler

	//	回收一个连接
	RecyclingConnection(handler client.ClientHandler)

	//	处理消息
	ProcessResponse(client.ClientHandler, int, []byte)

	//	关闭
	ProcessClose(handler client.ClientHandler)

	//	Close
	Close()
}

type connpool struct {
	idlec_mu sync.Mutex
	idlec    map[client.ClientHandler]struct{} //	闲置的连接
	hook_mu  sync.Mutex
	hook     map[client.ClientHandler]connection.TokenHandler //	连接与Token挂钩
}

func (this *connpool) Close() {
	this.idlec_mu.Lock()
	this.hook_mu.Lock()
	defer this.hook_mu.Unlock()
	defer this.idlec_mu.Unlock()
	for k, _ := range this.hook {
		this.idlec[k] = struct{}{}
		delete(this.hook, k)
	}
	for k, _ := range this.idlec {
		k.Close()
	}
	this.idlec = nil
	this.hook = nil

}

func (this *connpool) Connect(address string, count int) {
	this.idlec = make(map[client.ClientHandler]struct{})
	this.hook = make(map[client.ClientHandler]connection.TokenHandler)
	this.idlec_mu = sync.Mutex{}
	this.hook_mu = sync.Mutex{}
	for i := 0; i < count; i++ {
		qc := client.QClient{}
		qc.Dial(address, this.ProcessResponse, this.ProcessClose)
		this.idlec[&qc] = struct{}{}
	}
}

func (this *connpool) RecyclingConnection(handler client.ClientHandler) {
	this.idlec_mu.Lock()
	this.hook_mu.Lock()
	defer this.hook_mu.Unlock()
	defer this.idlec_mu.Unlock()

	this.idlec[handler] = struct{}{}
}

func (this *connpool) GetConnection(token connection.TokenHandler) client.ClientHandler {
	this.idlec_mu.Lock()
	this.hook_mu.Lock()
	defer this.hook_mu.Unlock()
	defer this.idlec_mu.Unlock()
	if len(this.idlec) == 0 {
		return nil
	}
	var res client.ClientHandler
	for k, _ := range this.idlec {
		res = k
		this.hook[res] = token
		break
	}
	return res
}

func (this *connpool) ProcessResponse(handler client.ClientHandler, n int, b []byte) {
	this.idlec_mu.Lock()
	this.hook_mu.Lock()
	defer this.hook_mu.Unlock()
	defer this.idlec_mu.Unlock()
	token := this.hook[handler]
	delete(this.hook, handler)
	this.idlec[handler] = struct{}{}
	token.Write(b)
}

func (this *connpool) ProcessClose(handler client.ClientHandler) {
	delete(this.idlec, handler)
}

func New(address string, count int) ConnectionPoolHandler {
	cp := connpool{}
	cp.Connect(address, count)
	return &cp
}
