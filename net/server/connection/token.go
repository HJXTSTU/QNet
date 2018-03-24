package connection

import (
	"net"
	"wwt/util"
	"sync"
	"log"
	"wwt/ctrl"
	"fmt"
)

const (
	BUFFER_SIZE = 1024
)

type ReadCallback func(TokenHandler, int, []byte)
type CloseCallback func(TokenHandler)
type SendCallback func(TokenHandler, []byte, int, error)

type TokenPoolHandler interface {
	AddToken(token TokenHandler)
	DeleteToken(token TokenHandler)
	CloseAll()
	Len() int
}

type TokenPool struct {
	tokens map[TokenHandler]TokenHandler
	mu     sync.Mutex
}

func (this *TokenPool)CloseAll(){
	this.mu.Lock()
	for _,v := range this.tokens{
		v.Close()
	}
	this.mu.Unlock()
}

func (this *TokenPool) AddToken(token TokenHandler) {
	this.mu.Lock()
	this.tokens[token] = token
	this.mu.Unlock()
}

func (this *TokenPool) DeleteToken(token TokenHandler) {
	this.mu.Lock()
	delete(this.tokens, token)
	this.mu.Unlock()
}

func (this *TokenPool) Len() int {
	this.mu.Lock()
	l := len(this.tokens)
	this.mu.Unlock()
	return l
}

func NewTokenPool() TokenPoolHandler {
	return &TokenPool{make(map[TokenHandler]TokenHandler), sync.Mutex{}}
}

type TokenHandler interface {
	ReadAsync()

	Close()

	OnRead(TokenHandler, int, []byte)

	OnClose(TokenHandler)

	RemoteAddr() net.Addr

	read(b []byte)(int,error)

	SendAsync(b []byte, callback SendCallback)
}

type QToken struct {
	conn     net.Conn
	onRead   ReadCallback
	onClose  CloseCallback
	r_stream util.StreamBuffer
}

func (this *QToken) SendAsync(b []byte, callback SendCallback) {
	n, err := this.conn.Write(b)
	if err == nil && callback != nil{
		callback(this, b, n, err)
	}else{
		log.Fatal(err)
	}
}

func (this *QToken)read(b []byte)(int,error){
	return this.conn.Read(b)
}

func (this *QToken) RemoteAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this *QToken)readAsync(handle TokenHandler){
	defer func() {
		err := recover().(error)
		if err.Error()=="EOF"{
			handle.OnClose(handle)
		}
	}()
	for {
		buf := make([]byte, BUFFER_SIZE)
		n, err := handle.read(buf)
		if n<=0 || err != nil{
			panic(err)
			break;
		}
		handle.OnRead(handle, n, buf[:n])
	}
	// TODO::err process

}

func (this *QToken) ReadAsync() {
	ctrl.StartGoroutines(func() {
		this.readAsync(this)
	})
}

func (this *QToken) OnRead(handle TokenHandler, n int, bytes []byte) {
	this.r_stream.Append(bytes)
	lstream := this.r_stream.Len()
	if lstream < 4 {
		return
	}
	length := this.r_stream.ReadInt()
	//	数据包已经完整
	if length <= lstream {
		data := this.r_stream.ReadNBytes(length)
		ctrl.StartGoroutines(func() {
			this.onRead(this, length, data)
		})
		//go this.onRead(this,length,data)
	} else {
		this.r_stream.Undo()
	}

}

func (this *QToken) OnClose(handle TokenHandler) {
	ctrl.StartGoroutines(func() {
		this.onClose(handle)
	})
	//go this.onClose(this)
}

func (this *QToken) Close() {
	fmt.Println("Close:",this.conn.RemoteAddr())
	this.conn.Close()
}

func NewQToken(conn net.Conn, onRead ReadCallback, onClose CloseCallback) *QToken {
	token := QToken{conn, onRead, onClose, util.NewStreamBuffer()}
	return &token
}
