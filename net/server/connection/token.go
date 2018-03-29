package connection

import (
	"net"
	"wwt/util"
	"sync"
	"wwt/ctrl"
)

const (
	BUFFER_SIZE = 32768
	RCHAN_SIZE  = 1024
	WCHAN_SIZE  = 1024
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
	for _, v := range this.tokens {
		v.Close()
	}
}

func (this *TokenPool) AddToken(token TokenHandler) {
	//	写入token 要加锁
	//	加锁操作O(1)的复杂度	不会引起其他阻塞
	this.mu.Lock()
	this.tokens[token] = token
	this.mu.Unlock()
}

func (this *TokenPool) DeleteToken(token TokenHandler) {
	//	删除token 要加锁
	//	删除操作O(1)的复杂度	不会引起其他阻塞
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

	RemoteAddr() net.Addr

	StartSend()

	Write([]byte)
}

type RChan chan []byte
type WChan chan []byte

type QToken struct {
	conn    net.Conn
	onRead  ReadCallback
	onClose CloseCallback

	r_exit   chan struct{}
	r_stream util.StreamBuffer
	r_chan   RChan

	w_exit chan struct{}
	w_chan WChan

	task_group sync.WaitGroup
	close_once sync.Once
}

func (this *QToken) Write(b []byte) {
	defer func() {
		_ = recover()
	}()
	select {
	case <-this.w_exit:
		return
	default:
		this.w_chan <- b
	}

}

func (this *QToken) sendAsync() {
	defer func() {
		this.task_group.Done()
		_ = recover() //	捕获异常	改层易触发conn close 异常
		this.Close()
	}()

	for {
		select {
		case <-this.w_exit:
			panic(nil)
			return
		case b := <-this.w_chan:
			if b != nil {
				stream := util.NewStreamBuffer()
				stream.WriteInt(len(b))
				stream.Append(b)
				n, err := this.conn.Write(stream.Bytes())
				if n <= 0 || err != nil {
					panic(err)
					return
				}
			} else {
				return
			}
		}
	}
}

func (this *QToken) StartSend() {
	ctrl.StartGoroutines(func() {
		this.sendAsync()
	})
}

func (this *QToken) read(b []byte) (int, error) {
	return this.conn.Read(b)
}

func (this *QToken) RemoteAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this *QToken) readAsync() {
	defer func() {
		this.task_group.Done()
		_ = recover()
		this.Close()
	}()

	for {
		select {
		case <-this.r_exit:
			panic(nil)
			return
		default:
			b := make([]byte, BUFFER_SIZE)
			n, err := this.conn.Read(b) //	可引发连接异常
			if n <= 0 || err != nil {
				panic(err)
				return
			}
			this.r_chan <- b[:n]
		}

	}
}

func (this *QToken) StartRead() {
	ctrl.StartGoroutines(func() {
		this.processRead()
	})
	ctrl.StartGoroutines(func() {
		this.readAsync()
	})
}

func (this *QToken) processRead() {
	defer func() {
		this.task_group.Done()
		_ = recover() //	捕获异常
		//	TODO::Nothing
	}()
	for {
		select {
		case <-this.r_exit:
			panic(nil)
			return
		case b := <-this.r_chan:
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
			} else {
				panic(nil)
				return
			}

		}
	}

}

func (this *QToken) Close() {
	this.close_once.Do(func() {
		close(this.r_exit) //	关闭对远端数据流的处理		影响到processRead方法		放弃从管道中读入数据并退出
		close(this.w_exit) //	对上层应用关闭输入口	影响到Write方法	针对准备写入数据时被阻塞的goroutine
		this.conn.Close()  //	关闭连接，readAsync,sendAsync会触发异常并退出
		//	清理w_chan
		ctrl.StartGoroutines(func() {
			for _ = range this.w_chan {

			}
		})

		//	清理r_chan
		ctrl.StartGoroutines(func() {
			for _ = range this.r_chan {

			}
		})
		this.task_group.Wait() //	等待该客户端所有任务	goroutuines	退出

		close(this.r_chan)     //	关闭处理数据流管道
		close(this.w_chan)     //	关闭发送数据流管道
		this.onClose(this)
	})

}

func NewQToken(conn net.Conn, onRead ReadCallback, onClose CloseCallback) *QToken {
	token := QToken{
		conn,
		onRead,
		onClose,
		make(chan struct{}),
		util.NewStreamBuffer(),
		make(RChan, RCHAN_SIZE),
		make(chan struct{}),
		make(WChan, WCHAN_SIZE),
		sync.WaitGroup{},
		sync.Once{},
	}
	token.task_group.Add(3)
	return &token
}
