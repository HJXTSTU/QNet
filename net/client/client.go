package client

import (
	"wwt/util"
	"log"
	"net"
	"wwt/ctrl"
	"sync"
	"fmt"
)

const (
	BUFFER_SIZE = 2048
	RCHAN_SIZE  = 128
	WCHAN_SIZE  = 128
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

	read(b []byte) (int, error)

	onClose()
}

type WChan chan []byte
type RChan chan []byte

type QClient struct {
	conn           net.Conn
	read_callback  ReadCallback
	close_callback CloseCallback

	r_chan   RChan
	r_stream util.StreamBuffer

	w_exit  chan struct{}
	w_chan  WChan
	w_group sync.WaitGroup
}

func (this *QClient) Close() {
	close(this.r_chan)
	this.conn.Close()

	close(this.w_exit)
	this.w_group.Wait()
	close(this.w_chan)
}

func (this *QClient) RemoteAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this *QClient) startSend() {
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

func (this *QClient) StartSend() {
	ctrl.StartGoroutines(func() {
		this.startSend()
	})
}

func (this *QClient) Write(b []byte) {
	select {
	case <-this.w_exit:
		return
	default:
		this.w_group.Add(1)
		this.w_chan <- b
		this.w_group.Done()
	}
}

func (this *QClient) Dial(address string, read_callback ReadCallback, close_callbcak CloseCallback) error {
	conn, err := net.Dial("tcp", address)
	if err == nil && conn != nil {
		this.conn = conn

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

func (this *QClient) readAsync(handler ClientHandler) {
	defer func() {
		err := recover().(error)
		if err.Error() == "EOF" {
			handler.onClose()
		}
	}()

	for {
		b := make([]byte, BUFFER_SIZE)
		n, err := handler.read(b)
		if n <= 0 || err != nil {
			panic(err)
			break
		}
		this.r_chan <- b[:n]
	}
}

func (this *QClient) processRead() {
	for b := range this.r_chan {
		if b != nil {
			this.r_stream.Append(b)
			for this.r_stream.Len() > 4 {
				length := this.r_stream.ReadInt()
				//fmt.Println(this.r_stream.Bytes())
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
			break
		}
	}

}

func (this *QClient) StartRead() {
	ctrl.StartGoroutines(func() {
		this.readAsync(this)
	})
	ctrl.StartGoroutines(func() {
		this.processRead()
	})
}

func (this *QClient) read(b []byte) (int, error) {
	return this.conn.Read(b)
}


func (this *QClient) onClose() {
	this.close_callback(this)
}
