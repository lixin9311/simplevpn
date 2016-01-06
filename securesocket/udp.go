package securesocket

import (
	"errors"
	"net"
	"sync"
)

type PacketConn struct {
	rlock sync.Mutex
	wlock sync.Mutex
	net.PacketConn
	*Cipher
	readBuf  []byte
	writeBuf []byte
}

func NewPacketConn(c net.PacketConn, cipher *Cipher) *PacketConn {
	return &PacketConn{
		PacketConn: c,
		Cipher:     cipher,
		readBuf:    leakyBuf.Get(),
		// for thread safety
		//writeBuf: leakyBuf.Get(),
	}
}

func (c *PacketConn) Close() error {
	leakyBuf.Put(c.readBuf)
	//leakyBuf.Put(c.writeBuf)
	return c.PacketConn.Close()
}

func (c *PacketConn) ReadFrom(b []byte) (n int, src net.Addr, err error) {
	c.rlock.Lock()
	defer c.rlock.Unlock()
	n, src, err = c.PacketConn.ReadFrom(c.readBuf[0:])
	if err != nil {
		return
	}
	if n < c.info.ivLen {
		return 0, nil, errors.New("[Packet]read error: cannot decrypt")
	}
	iv := make([]byte, c.info.ivLen)
	copy(iv, c.readBuf[:c.info.ivLen])
	if err = c.initDecrypt(iv); err != nil {
		return
	}
	c.decrypt(b[0:n-c.info.ivLen], c.readBuf[c.info.ivLen:n])
	n = n - c.info.ivLen
	return
}

func (c *PacketConn) WriteTo(b []byte, dst net.Addr) (n int, err error) {
	c.wlock.Lock()
	defer c.wlock.Unlock()
	dataStart := 0

	var iv []byte
	iv, err = c.initEncrypt()
	if err != nil {
		return
	}
	// Put initialization vector in buffer, do a single write to send both
	// iv and data.
	cipherData := make([]byte, len(b)+len(iv))
	copy(cipherData, iv)
	dataStart = len(iv)

	c.encrypt(cipherData[dataStart:], b)
	n, err = c.PacketConn.WriteTo(cipherData, dst)
	return
}
