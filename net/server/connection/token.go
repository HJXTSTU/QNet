package connection

import (
	"net"
	"wwt/util"
	"sync"
	"wwt/ctrl"
	"fmt"
)

const (
	BUFFER_SIZE = 32768
	RCHAN_SIZE  = 128
	WCHAN_SIZE  = 128
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

func (this *TokenPool) CloseAll() {
	this.mu.Lock()
	for _, v := range this.tokens {
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
	defer this.mu.Unlock()
	return l
}

func NewTokenPool() TokenPoolHandler {
	return &TokenPool{make(map[TokenHandler]TokenHandler), sync.Mutex{}}
}

type TokenHandler interface {
	StartRead()

	Close()

	OnClose(TokenHandler)

	RemoteAddr() net.Addr

	StartSend()

	Write([]byte)

	read(b []byte) (int, error)
}

type RChan chan []byte
type WChan chan []byte

type QToken struct {
	conn     net.Conn
	onRead   ReadCallback
	onClose  CloseCallback
	r_stream util.StreamBuffer
	r_chan   RChan

	w_exit  chan struct{}
	w_chan  WChan
	w_group sync.WaitGroup
}

func (this *QToken) Write(b []byte) {
	select {
	case <-this.w_exit:
		return
	default:
		this.w_group.Add(1)
		this.w_chan <- b
		this.w_group.Done()
	}

}

func (this *QToken) startSend() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err.(error))
		}
	}()
	for b := range this.w_chan {
		if b != nil {
			stream := util.NewStreamBuffer()
			stream.WriteInt(len(b))
			stream.Append(b)
			n, err := this.conn.Write(stream.Bytes())
			if n <= 0 || err != nil {
				panic(err)
				break
			}
		} else {
			break
		}
	}
}

func (this *QToken) StartSend() {
	ctrl.StartGoroutines(func() {
		this.startSend()
	})
}

func (this *QToken) read(b []byte) (int, error) {
	return this.conn.Read(b)
}

func (this *QToken) RemoteAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this *QToken) readAsync(handle TokenHandler) {
	defer func() {
		err := recover().(error)
		if err.Error() == "EOF" {
			handle.OnClose(handle)
		}
	}()

	for {
		buf := make([]byte, BUFFER_SIZE)
		n, err := handle.read(buf)
		if n <= 0 || err != nil {
			panic(err)
			break;
		}
		//fmt.Println(buf[:n])
		this.r_chan <- buf[:n]
	}
	// TODO::err process
}

func (this *QToken) StartRead() {
	ctrl.StartGoroutines(func() {
		this.processRead()
	})
	ctrl.StartGoroutines(func() {
		this.readAsync(this)
	})
}

func (this *QToken) processRead() {
	for b := range this.r_chan {
		if b != nil {
			this.r_stream.Append(b)
			for this.r_stream.Len() > 4 {
				length := this.r_stream.ReadInt()
				//	数据包已经完整
				if !this.r_stream.Empty() && length <= this.r_stream.Len() {
					data := this.r_stream.ReadNBytes(length)
					this.onRead(this, length, data)
				} else {
					this.r_stream.Undo()
					break
				}
			}
			if this.r_stream.Empty() {
				this.r_stream.Renew()
			}
		} else {
			break
		}
	}

}

func (this *QToken) OnClose(handle TokenHandler) {
	ctrl.StartGoroutines(func() {
		this.onClose(handle)
	})
}

func (this *QToken) Close() {
	fmt.Println("Close:", this.conn.RemoteAddr())
	close(this.r_chan)
	this.conn.Close()

	close(this.w_exit)
	this.w_group.Wait()
	close(this.w_chan)
}

func NewQToken(conn net.Conn, onRead ReadCallback, onClose CloseCallback) *QToken {
	token := QToken{
		conn,
		onRead,
		onClose,
		util.NewStreamBuffer(),
		make(RChan, RCHAN_SIZE),
		make(chan struct{}),
		make(WChan, WCHAN_SIZE),
		sync.WaitGroup{}}
	return &token
}
