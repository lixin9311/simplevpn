package securesocket

import (
	"bytes"
	"net"
	"testing"
)

func TestSecureTCP(t *testing.T) {
	data := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	port := ":0"
	tcpAddr, err := net.ResolveTCPAddr("tcp", port)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		t.Fatal(err)
	}
	Addr := ln.Addr()
	defer ln.Close()
	cipher, err := NewCipher("aes-256-cfb", "password")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		conn, err := ln.AcceptTCP()
		if err != nil {
			t.Fatal(err)
		}
		secureConn := NewConn(conn, cipher.Copy())
		buf := make([]byte, 1500)
		n, err := secureConn.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buf[:n], data) {
			t.Error("TCP server recieved data does not match.")
		}
		_, err = secureConn.Write(buf[:n])
		if err != nil {
			t.Fatal(err)
		}
		return
	}()
	host, port, err := net.SplitHostPort(Addr.String())
	if err != nil {
		t.Fatal(err)
	}
	host = "localhost"
	conn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		t.Fatal(err)
	}
	secureconn := NewConn(conn.(*net.TCPConn), cipher.Copy())
	_, err = secureconn.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1500)
	n, err := secureconn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf[:n], data) {
		t.Error("TCP client read does not match.")
	}

}
