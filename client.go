package main

import (
	"log"
    "net"
	"github.com/lixin9311/simplevpn/tap"
	ss "github.com/lixin9311/simplevpn/securesocket"
    "time"
)

func main() {
	ifce, err := tap.NewTAP()
    if err != nil {
        log.Fatalln("Failed to create TAP interface:", err)
    }
    defer ifce.Close()
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
    go func(){
        buf := make([]byte, 1522)
        for {
            n, _ := ifce.Read(buf)
            secureconn.Write(buf[:n])
        }
    }()
    go func(){
        secureconn.Write([]byte("Beep"))
        time.Sleep(time.Second)
    }()
    buf := make([]byte, 1522)
    for {
        n, _ := secureconn.Read(buf)
        log.Println("Read:", buf[:n])
        ifce.Write(buf[:n])
    }
}

