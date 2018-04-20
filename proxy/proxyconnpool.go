package proxy

import (
	"wwt/util"
	"wwt/net/client"
	"log"
	"wwt/net/server/connection"
)

type ResponseCallback func(token connection.TokenHandler, n int, b []byte)

type ProxyHandle interface {
	//	增加被代理者
	AddPrincipal(principal connection.TokenHandler)

	//	建立连接
	Connect(addr string, count int, callback ResponseCallback)

	//	处理远程主机的返回消息
	ProcessRemoteMessage(handler client.ClientHandler, n int, b []byte)

	//	处理远程主机关闭的消息
	ProcessClose(handler client.ClientHandler)

	//	回收连接
	RecyclingConnection(principal connection.TokenHandler)

	//	查看是否有连接
	HasPrincipal(principal connection.TokenHandler) bool

	//	处理代理消息
	ProcessProxyMessage(k connection.TokenHandler, stream util.StreamBuffer)

	//	关闭代理连接
	Close()
}

type qproxy struct {
	//	空闲连接
	idlec *util.QMap

	//	被使用的连接
	hook *util.QMap

	//	逆向连接
	nhook *util.QMap

	//	远程服务器地址
	remote_addr string

	//	回调远程消息函数
	response_callback ResponseCallback
}

func (this *qproxy) Close() {
	for this.nhook.Length() > 0 {
		c := this.nhook.GetAnyValue().(client.ClientHandler)
		t := this.hook.Get(c).(connection.TokenHandler)
		this.nhook.Delete(t)
		this.hook.Delete(c)
		c.Close()
	}
	for this.idlec.Length() > 0 {
		c := this.idlec.GetAnyKey().(client.ClientHandler)
		c.Close()
	}
}

func (this *qproxy) HasPrincipal(principal connection.TokenHandler) bool {
	if this.nhook.HasKey(principal) {
		return true
	} else {
		return false
	}
}

func (this *qproxy) ProcessProxyMessage(principal connection.TokenHandler, stream util.StreamBuffer) {
	if !this.nhook.HasKey(principal) {
		this.AddPrincipal(principal)
	}
	c := this.nhook.Get(principal).(client.ClientHandler)
	c.Write(stream.Bytes())
}

func (this *qproxy) RecyclingConnection(principal connection.TokenHandler) {
	c := this.nhook.Get(principal)
	this.nhook.Delete(principal)
	this.hook.Delete(c)
	this.idlec.Set(c, struct{}{})
}

func (this *qproxy) AddPrincipal(principal connection.TokenHandler) {
	if this.idlec.Length() == 0 {
		nc := this.newConnection()
		this.idlec.Set(nc, struct{}{})
	}
	c := this.idlec.GetAnyKey()
	this.idlec.Delete(c)
	this.hook.Set(c, principal)
	this.nhook.Set(principal, c)
}

func (this *qproxy) ProcessRemoteMessage(c client.ClientHandler, n int, b []byte) {
	token := this.hook.Get(c).(connection.TokenHandler)
	this.response_callback(token, n, b)
}

func (this *qproxy) ProcessClose(handler client.ClientHandler) {
	if this.idlec.HasKey(handler) {
		this.idlec.Delete(handler)
	}
	var token connection.TokenHandler
	if this.hook.HasKey(handler) {
		token = this.hook.Get(handler).(connection.TokenHandler)
		this.hook.Delete(handler)
	}
	if this.nhook.HasKey(token) {
		this.nhook.Delete(token)
	}
}

func (this *qproxy) Connect(addr string, count int, callback ResponseCallback) {
	this.remote_addr = addr
	cnt := 0
	this.idlec = util.NewQMap()
	this.hook = util.NewQMap()
	this.nhook = util.NewQMap()
	for i := 0; i < count; i++ {
		c := this.newConnection()
		if c != nil {
			this.idlec.Set(c, struct{}{})
			cnt++
		}

	}
	this.response_callback = callback
	log.Printf("QProxy.Connect: make %d idle connections.\n", cnt)
}

func (this *qproxy) newConnection() client.ClientHandler {
	c := client.QClient{}
	err := c.Dial(this.remote_addr, this.ProcessRemoteMessage, this.ProcessClose)
	if err != nil {
		log.Printf("QProxy.newConnection: create connection fail. %s.\n", err.Error())
		return nil
	}
	return &c
}

func NewProxy() ProxyHandle {
	return &qproxy{}
}
