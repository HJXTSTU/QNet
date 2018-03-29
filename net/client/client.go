package client

import (
	"wwt/util"
	"log"
	"net"
	"wwt/ctrl"
	"sync"
)

const (
	BUFFER_SIZE = 32768
	RCHAN_SIZE  = 1024
	WCHAN_SIZE  = 1024
)

type ReadCallback func(ClientHandler, int, []byte)
type CloseCallback func(ClientHandler)
type SendCallback func(ClientHandler, []byte, int, error)

type ClientHandler interface {
	Dial(address string, read_callback ReadCallback, close_callbcak CloseCallback) error

	StartRead()

	StartSend()

	Write([]byte)

	Close()

	RemoteAddr() net.Addr
}

type WChan chan []byte
type RChan chan []byte

type QClient struct {
	conn           net.Conn
	read_callback  ReadCallback
	close_callback CloseCallback

	r_exit   chan struct{}
	r_chan   RChan
	r_stream util.StreamBuffer

	w_exit chan struct{}
	w_chan WChan

	task_group sync.WaitGroup
	close_once sync.Once
}

func (this *QClient) Close() {
	this.close_once.Do(func() {
		close(this.r_exit) //	关闭对远端数据流的处理		影响到processRead方法		放弃从管道中读入数据并退出
		close(this.w_exit) //	对上层应用关闭输入口	影响到Write方法	针对准备写入数据时被阻塞的goroutine

		this.conn.Close() //	关闭连接，readAsync,sendAsync会触发异常并退出

		this.task_group.Wait() //	等待该客户端所有任务	goroutuines	退出

		close(this.r_chan) //	关闭处理数据流管道
		close(this.w_chan) //	关闭发送数据流管道
		this.close_callback(this)
	})

}

func (this *QClient) RemoteAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this *QClient) sendAsync() {
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

func (this *QClient) StartSend() {
	ctrl.StartGoroutines(func() {
		this.sendAsync()
	})
}

func (this *QClient) Write(b []byte) {
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

func (this *QClient) Dial(address string, read_callback ReadCallback, close_callbcak CloseCallback) error {
	conn, err := net.Dial("tcp", address)
	if err == nil && conn != nil {
		this.conn = conn
		this.task_group.Add(3)
		this.r_exit = make(chan struct{})
		this.r_chan = make(RChan, RCHAN_SIZE)
		this.r_stream = util.NewStreamBuffer()
		this.read_callback = read_callback
		this.StartRead()

		this.close_callback = close_callbcak

		this.w_exit = make(chan struct{})
		this.w_chan = make(WChan, WCHAN_SIZE)
		this.StartSend()
		log.Printf("Connect to %s.\n", conn.RemoteAddr().String())
		return nil
	} else {
		return err
	}
}

func (this *QClient) readAsync() {
	defer func() {
		this.task_group.Done()
		_ = recover()
		this.Close()
	}()
	b := make([]byte, BUFFER_SIZE)
	for {
		select {
		case <-this.r_exit:
			panic(nil)
			return
		default:

			n, err := this.conn.Read(b) //	可引发连接异常
			if n <= 0 || err != nil {
				panic(err)
				return
			}
			this.r_chan <- b[:n]
		}
	}
}

func (this *QClient) processRead() {
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
						this.read_callback(this, length, data)
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

func (this *QClient) StartRead() {

	ctrl.StartGoroutines(func() {
		this.readAsync()
	})
	ctrl.StartGoroutines(func() {
		this.processRead()
	})
}

func (this *QClient) read(b []byte) (int, error) {
	return this.conn.Read(b)
}
