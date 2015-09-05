package main

import (
	"log"
    "net"
	"time"
	ss "github.com/lixin9311/simplevpn/securesocket"
)

func main() {
    conn, err := net.Dial("udp", "6.lucus.moe:9023")
    if err != nil {
        log.Fatalln("Failed to dial:", err)
    }
    cipher, err := ss.NewCipher("aes-256-cfb", "lixin93")
    if err != nil {
        log.Fatalln("Failed to create cipher:", err)
    }
    secureconn := ss.NewUDPConn(conn.(*net.UDPConn), cipher.Copy())
    defer secureconn.Close()
    for {
		buf := make([]byte, 10)
		buf[0] = 1
		time.Sleep(time.Second)
		secureconn.Write(buf[:5])
        n, _ := secureconn.Read(buf)
        log.Println("Read:", buf[:n])
    }
}

