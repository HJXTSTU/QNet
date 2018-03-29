package server

import (
	"wwt/net/server/listener"
	"wwt/net/server/connection"
	"net"
	"log"
)

type QServerHandle interface {

	AsyncListen()

	SyncListen()

	Close()

	SetProcesser(ProcesseFunc)
}

type ProcesseFunc	func(connection.TokenHandler, int, []byte)

type QWriter interface {
	Send([]byte)
}

type QServer struct {
	listener  listener.ListenerHandle
	tokens    connection.TokenPoolHandler
	processeFunc ProcesseFunc
}

func (this *QServer)Close(){
	this.tokens.CloseAll()
	this.listener.Close()
}

func (this *QServer) AsyncListen() {
	this.listener.AsyncAccept(this.onAccept)
}

func (this *QServer) SyncListen() {
	this.listener.SyncAccept(this.onAccept)
}

func (this *QServer) onAccept(conn net.Conn) {
	token := connection.NewQToken(conn, this.onRead, this.onClose)
	this.tokens.AddToken(token)
	token.StartRead()
	token.StartSend()
	log.Printf("QServer %p: New connection enter %s. Create token %p.\n",this,token.RemoteAddr(),&token)
}

func (this *QServer) onRead(handle connection.TokenHandler, n int, bytes []byte) {
	this.processeFunc(handle, n, bytes)
}

func (this *QServer) SetProcesser(p ProcesseFunc) {
	this.processeFunc = p
}

func (this *QServer) onClose(handle connection.TokenHandler) {
	//TODO::关闭TOKEN
	//handle.Close()
	this.tokens.DeleteToken(handle)
	this.listener.ReleaseConn()
	log.Printf("QServer %p: Delete token %p. Remain: %d.\n",this,&handle,this.tokens.Len())
	//fmt.Println("Remain:",this.tokens.Len())
}

func NewQServer(address string) QServerHandle {
	qserver := new(QServer)
	qserver.listener = listener.NewListener(address)
	qserver.tokens = connection.NewTokenPool()
	return qserver
}
