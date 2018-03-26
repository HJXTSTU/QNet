package util

import (
	"math"
	"errors"
)

func bytesToUint16(p []byte) uint16 {
	var bits uint16
	for _, v := range p {
		bits <<= 8
		bits |= uint16(v)
	}
	return bits
}

func bytesToUint32(p []byte) uint32 {
	var bits uint32 = 0
	for _, v := range p {
		bits <<= 8
		bits |= uint32(v)
	}
	return bits
}

func bytesToUint64(p []byte) uint64 {
	var bits uint64 = 0
	for _, v := range p {
		bits <<= 8
		bits |= uint64(v)
	}
	return bits
}

func bytesToInt(p []byte) int {
	var bits int32 = 0
	for _, v := range p {
		bits <<= 8
		bits |= int32(v)
	}
	return int(bits)
}

func bytesToFloat32(p []byte) float32 {
	return math.Float32frombits(bytesToUint32(p))
}

func bytesToFloat64(p []byte) float64 {
	return math.Float64frombits(bytesToUint64(p))
}

func uint16ToBytes(n uint16) []byte {
	bytes := make([]byte, 2)
	bytes[0] = byte(n >> 8)
	bytes[1] = byte(n)
	return bytes
}

func uint32ToBytes(n uint32) []byte {
	bytes := make([]byte, 4)
	bytes[0] = byte(n >> 24)
	bytes[1] = byte(n >> 16)
	bytes[2] = byte(n >> 8)
	bytes[3] = byte(n)
	return bytes
}

func uint64ToBytes(n uint64) []byte {
	bytes := make([]byte, 8)
	bytes[0] = byte(n >> 56)
	bytes[1] = byte(n >> 48)
	bytes[2] = byte(n >> 40)
	bytes[3] = byte(n >> 32)
	bytes[4] = byte(n >> 24)
	bytes[5] = byte(n >> 16)
	bytes[6] = byte(n >> 8)
	bytes[7] = byte(n)
	return bytes
}

func intToBytes(i int) []byte {
	bytes := make([]byte, 4)
	n := int32(i)
	bytes[0] = byte(n >> 24)
	bytes[1] = byte(n >> 16)
	bytes[2] = byte(n >> 8)
	bytes[3] = byte(n)
	return bytes
}

func float32ToBytes(n float32) []byte {
	return uint32ToBytes(math.Float32bits(n))
}

func float64ToBytes(n float64) []byte {
	return uint64ToBytes(math.Float64bits(n))
}

type StreamBuffer interface {
	Append([]byte)

	Bytes() []byte

	ReadByte() byte
	WriteByte(byte)

	ReadInt() int;
	WriteInt(n int)

	ReadFloat32() float32
	WriteFloat32(float32)

	ReadFloat64() float64
	WriteFloat64(float64)

	ReadLine() string
	WriteLine(str string)

	ReadNBytes(n int) []byte
	WriteNBytes(b []byte, n int)

	Write(p []byte) (n int, err error)

	Renew()

	Reset()

	Len() int

	Empty() bool

	Undo()
}

type stream struct {
	buf []byte
	off int
	cur int

	undoOffset int
}

func (this *stream) mem() {
	this.undoOffset = this.cur
}

func (this *stream) ReadLine() string {
	endl := this.cur
	for this.buf[endl] != '\n' && endl < this.off {
		endl++
	}
	start := this.cur
	this.mem()
	this.cur = endl;
	if this.cur != this.off && this.buf[this.cur] == '\n' {
		this.cur++
	}
	return string(this.buf[start:endl])
}

func (this *stream) WriteLine(str string) {
	bytes := []byte(str)
	this.Append(bytes)
	this.WriteByte('\n')
}

func (this *stream) Append(p []byte) {
	this.buf = append(this.buf, p...)
	this.off += len(p)
}

func (this *stream) Bytes() []byte {
	return this.buf[this.cur:this.off]
}

func (this *stream) Len() int {
	return this.off - this.cur
}

func (this *stream) Empty() bool {
	return this.off == this.cur
}

func (this *stream) ReadByte() byte {
	res := this.buf[this.cur]
	this.mem()
	this.cur++
	return res
}
func (this *stream) WriteByte(b byte) {
	this.buf = append(this.buf, b)
	this.off++
}

func (this *stream) ReadInt() int {
	if this.Empty() {
		panic(errors.New("Stream is empty."))
	}

	//fmt.Printf("Current:cur:%d\toff:%d\n", this.cur, this.off)
	if this.cur+4 > this.off || this.cur < 0 || this.off < 0 {
		panic(nil)
	}
	p := this.buf[this.cur:this.cur+4]
	i := bytesToInt(p)
	this.mem()
	this.cur += 4
	return i
}

func (this *stream) WriteInt(n int) {
	this.buf = append(this.buf, intToBytes(n)...)
	this.off += 4
}

func (this *stream) ReadFloat32() float32 {
	if this.Empty() {
		panic(errors.New("Stream is empty."))
	}
	p := this.buf[this.cur:this.cur+4]
	f := bytesToFloat32(p)
	this.mem()
	this.cur += 4
	return f
}

func (this *stream) WriteFloat32(f float32) {
	this.buf = append(this.buf, float32ToBytes(f)...)
	this.off += 4
}

func (this *stream) ReadFloat64() float64 {
	if this.Empty() {
		panic(errors.New("Stream is empty."))
	}
	p := this.buf[this.cur:this.cur+8]
	f := bytesToFloat64(p)
	this.mem()
	this.cur += 8
	return f
}

func (this *stream) WriteFloat64(f float64) {
	this.buf = append(this.buf, float64ToBytes(f)...)
	this.off += 8
}

func (this *stream) Write(p []byte) (n int, err error) {
	l := len(p)
	this.buf = append(this.buf, p...)
	this.off += l
	return l, nil
}

func (this *stream) Renew() {
	this.buf = make([]byte, 0)
	this.cur = 0
	this.off = 0
	this.undoOffset = 0;
}

func (this *stream) Undo() {
	this.cur = this.undoOffset
}

func (this *stream) ReadNBytes(n int) []byte {
	cur := this.cur
	this.mem()
	this.cur += n
	if this.cur == this.off {
		defer this.Renew()
	}
	res := make([]byte, n)
	copy(res, this.buf[cur:cur+n])
	//fmt.Printf("Current:cur:%d\toff:%d\n", this.cur, this.off)
	return res
}

func (this *stream) WriteNBytes(b []byte, n int) {
	if len(b) < n {
		n = len(b)
	}
	this.Append(b[:n])
}

//	重置stream的cur下标
func (this *stream) Reset() {
	this.cur = 0
	this.undoOffset = 0;
	//this.off = 0
}

func NewStreamBuffer() StreamBuffer {
	return &stream{make([]byte, 0), 0, 0, 0}
}
