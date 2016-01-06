package securesocket

import (
	"errors"
	"io"
	"log"
	"net"
)

type PacketConn struct {
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

type PacketConn_Listener struct {
	net.PacketConn
	children map[net.Addr]*PacketConn_Conn
	fork     chan *PacketConn_Conn
}

func NewPacketConn_Listener(c net.PacketConn) *PacketConn_Listener {
	ln := &PacketConn_Listener{PacketConn: c, children: map[net.Addr]*PacketConn_Conn{}, fork: make(chan *PacketConn_Conn, 8)}
	ln.init()
	return ln
}
func (ln *PacketConn_Listener) init() {
	go func() {
		buf := make([]byte, 1600)
		for {
			n, addr, err := ln.ReadFrom(buf)
			if err != nil {
				log.Println("Failed to read from fake listener:", err)
				break
			}
			if v, ok := ln.children[addr]; !ok {
				c := &PacketConn_Conn{PacketConn: ln, remoteAddr: addr, input: make(chan []byte, 8)}
				ln.children[addr] = c
				c.input <- buf[:n]
				ln.fork <- c
			} else {
				v.input <- buf[:n]
			}
		}
	}()
}

func (ln *PacketConn_Listener) Accept() (c net.Conn, err error) {
	c = <-ln.fork
	return
}

func (ln *PacketConn_Listener) Addr() net.Addr {
	return ln.PacketConn.LocalAddr()
}

type PacketConn_Conn struct {
	// write only
	net.PacketConn
	remoteAddr net.Addr
	input      chan []byte
}

func (c *PacketConn_Conn) Read(b []byte) (n int, err error) {
	buf := <-c.input
	if len(b) < len(buf) {
		log.Println("Buf is too short.")
		return n, io.ErrShortBuffer
	}
	copy(b, buf)
	n = len(buf)
	return
}
func (c *PacketConn_Conn) Write(b []byte) (n int, err error) {
	return c.PacketConn.WriteTo(b, c.remoteAddr)
}
func (c *PacketConn_Conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}
