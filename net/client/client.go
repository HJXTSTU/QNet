package client

import (
	"wwt/util"
	"log"
	"net"
	"wwt/ctrl"
)

const (
	BUFFER_SIZE = 1024
)

type ReadCallback func(ClientHandler, int, []byte)
type CloseCallback func(ClientHandler)
type SendCallback func(ClientHandler, []byte, int, error)

type ClientHandler interface {
	Connect(address string, read_callback ReadCallback, close_callbcak CloseCallback) error

	SendAsync([]byte, SendCallback)

	ReadAsync()

	Close()

	RemoteAddr() net.Addr

	read(b []byte) (int, error)

	onRead(b []byte)

	onClose()
}

type QClient struct {
	conn           net.Conn
	r_stream       util.StreamBuffer
	read_callback  ReadCallback
	close_callback CloseCallback
}

func (this *QClient) Close() {
	this.conn.Close()
}

func (this *QClient) RemoteAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this *QClient) Connect(address string, read_callback ReadCallback, close_callbcak CloseCallback) error {
	conn, err := net.Dial("tcp", address)
	if err == nil && conn != nil {
		this.conn = conn
		this.r_stream = util.NewStreamBuffer()
		this.read_callback = read_callback
		this.close_callback = close_callbcak
		log.Printf("Connect to %s.\n", conn.RemoteAddr().String())
		return nil
	} else {
		return err
	}
}

func (this *QClient) readAsync(handler ClientHandler) {
	for {
		b := make([]byte, BUFFER_SIZE)
		n, err := handler.read(b)
		if n <= 0 || err != nil {
			break
		}
		handler.onRead(b[:n])
	}
	handler.onClose()
}

func (this *QClient) ReadAsync() {
	ctrl.StartGoroutines(func() {
		this.readAsync(this)
	})
	//go func(handler ClientHandler) {
	//	for {
	//		b := make([]byte, BUFFER_SIZE)
	//		n, err := handler.read(b)
	//		if n <= 0 || err != nil {
	//			break
	//		}
	//		handler.onRead(b[:n])
	//	}
	//	handler.onClose()
	//}(this)
}

func (this *QClient) read(b []byte) (int, error) {
	return this.conn.Read(b)
}

func (this *QClient) onRead(b []byte) {
	this.r_stream.Append(b)
	lstream := this.r_stream.Len()
	if lstream < 4 {
		return
	}
	lmsg := this.r_stream.ReadInt()
	if lmsg <= lstream {
		data := this.r_stream.ReadNBytes(lmsg)
		ctrl.StartGoroutines(func() {
			this.read_callback(this, lmsg, data)
		})
		//go this.read_callback(this, lmsg, data)
	} else {
		this.r_stream.Undo()
	}

}

func (this *QClient) SendAsync(b []byte, callback SendCallback) {
	n, err := this.conn.Write(b)
	if err == nil {
		if callback != nil {
			ctrl.StartGoroutines(func() {
				callback(this, b, n, nil)
			})
			//go callback(this, b, n, nil)
		}
	} else {
		log.Fatal(err)
	}
}

func (this *QClient) onClose() {
	//ctrl.StartGoroutines(func() {
	this.close_callback(this)
	//}
	//go this.close_callback(this)
}
