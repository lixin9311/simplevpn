package securesocket

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
)

type Conn struct {
	declock sync.Mutex
	enclock sync.Mutex
	net.Conn
	*Cipher
	readBuf  []byte
	writeBuf []byte
}

func NewConn(c net.Conn, cipher *Cipher) *Conn {
	return &Conn{
		Conn:     c,
		Cipher:   cipher,
		readBuf:  leakyBuf.Get(),
		writeBuf: leakyBuf.Get(),
	}
}

func (c *Conn) Close() error {
	leakyBuf.Put(c.readBuf)
	leakyBuf.Put(c.writeBuf)
	return c.Conn.Close()
}

func (c *Conn) Read(b []byte) (n int, err error) {
	c.declock.Lock()
	defer c.declock.Unlock()
	if c.dec == nil {
		iv := make([]byte, c.info.ivLen)
		if _, err = io.ReadFull(c.Conn, iv); err != nil {
			return
		}
		if err = c.initDecrypt(iv); err != nil {
			return
		}
	}

	cipherData := c.readBuf
	if len(b) > len(cipherData) {
		cipherData = make([]byte, len(b))
	} else {
		cipherData = cipherData[:len(b)]
	}

	n, err = c.Conn.Read(cipherData)
	if n > 0 {
		c.decrypt(b[0:n], cipherData[0:n])
	}
	return
}

func (c *Conn) Write(b []byte) (n int, err error) {
	c.enclock.Lock()
	defer c.enclock.Unlock()
	var iv []byte
	if c.enc == nil {
		iv, err = c.initEncrypt()
		if err != nil {
			return
		}
	}

	cipherData := c.writeBuf
	dataSize := len(b) + len(iv)
	if dataSize > len(cipherData) {
		cipherData = make([]byte, dataSize)
	} else {
		cipherData = cipherData[:dataSize]
	}

	if iv != nil {
		// Put initialization vector in buffer, do a single write to send both
		// iv and data.
		copy(cipherData, iv)
	}
	c.encrypt(cipherData[len(iv):], b)
	n, err = c.Conn.Write(cipherData)
	return
}

type PacketStreamConn struct {
	net.Conn
}

func NewPacketStreamConn(c net.Conn) *PacketStreamConn {
	return &PacketStreamConn{c}
}

func (c *PacketStreamConn) Write(b []byte) (n int, err error) {
	var size uint16
	var tmp [2]byte
	size = uint16(len(b))
	binary.BigEndian.PutUint16(tmp[:], size)
	n, err = c.Conn.Write(append(tmp[:], b...))
	n -= 2
	return
}

func (c *PacketStreamConn) Read(b []byte) (n int, err error) {
	var size uint16
	var tmp [2]byte
	n, err = io.ReadFull(c.Conn, tmp[:2])
	if err != nil {
		return 0, err
	}
	size = binary.BigEndian.Uint16(tmp[:])
	if int(size) > len(b) {
		return 0, io.ErrShortBuffer
	}
	n, err = io.ReadFull(c.Conn, b[:int(size)])
	return
}
