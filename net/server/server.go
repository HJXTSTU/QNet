package server

import (
	"wwt/net/server/listener"
	"wwt/net/server/connection"
	"net"
	"fmt"
)

type QServerHandle interface {

	AsyncListen()

	SyncListen()

	Close()

	SetProcesser(QProcesser)
}

type QProcesser interface {
	Processe(connection.TokenHandler, int, []byte)
}

type QWriter interface {
	Send([]byte)
}

type QServer struct {
	listener  listener.ListenerHandle
	tokens    connection.TokenPoolHandler
	processer QProcesser
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
	token.ReadAsync()
	//this.listener.AcceptAsync(this.onAccept)
}

func (this *QServer) onRead(handle connection.TokenHandler, n int, bytes []byte) {
	this.processer.Processe(handle, n, bytes)
}

func (this *QServer) SetProcesser(p QProcesser) {
	this.processer = p
}

func (this *QServer) onClose(handle connection.TokenHandler) {
	//TODO::关闭TOKEN
	handle.Close()
	this.tokens.DeleteToken(handle)
	this.listener.ReleaseConn()
	fmt.Println("Remain:",this.tokens.Len())

}

func NewQServer(address string) QServerHandle {
	qserver := new(QServer)
	qserver.listener = listener.NewListener(address)
	qserver.tokens = connection.NewTokenPool()
	return qserver
}
