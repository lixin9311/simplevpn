package securesocket

import (
	"encoding/binary"
	"errors"
)

type TCPConn_packet_safe struct {
	*TCPConn
	readBuf  []byte
	writeBuf []byte
}

func NewPacketSafeTCPConn(c *TCPConn) *TCPConn_packet_safe {
	return &TCPConn_packet_safe{
		TCPConn:  c,
		readBuf:  leakyBuf.Get(),
		writeBuf: leakyBuf.Get(),
	}
}

func (self *TCPConn_packet_safe) Close() error {
	leakyBuf.Put(self.readBuf)
	leakyBuf.Put(self.writeBuf)
	return self.TCPConn.Close()
}

func (self *TCPConn_packet_safe) Read(b []byte) (n int, err error) {
	headerBuf := make([]byte, 4)
	n, err = self.TCPConn.Read(headerBuf)
	if err != nil {
		return
	}
	length := binary.BigEndian.Uint32(headerBuf)
	n, err = self.TCPConn.Read(b[:length])
	if err != nil {
		return
	}
	if n != int(length) {
		err = errors.New("Read length mismatched!")
		return
	}
	return
}

func (self *TCPConn_packet_safe) Write(b []byte) (n int, err error) {
	length := len(b)
	binary.BigEndian.PutUint32(self.writeBuf, uint32(length))
	n = copy(self.writeBuf[4:], b)
	return self.TCPConn.Write(self.writeBuf[:n+4])
}
