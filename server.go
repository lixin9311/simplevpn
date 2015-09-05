package main

import (
	"log"
    "net"
	"github.com/lixin9311/simplevpn/tap"
	ss "github.com/lixin9311/simplevpn/securesocket"
)

func handleConn(c *ss.TCPConn, ifce *tap.Interface) {
    go ss.Pipe(c, ifce)
    ss.Pipe(ifce, c)
}

var mainaddr net.Addr

func main() {
	ifce, err := tap.NewTAP()
    if err != nil {
        log.Fatalln("Failed to create TAP interface:", err)
    }
    defer ifce.Close()
    ln, err := net.ListenPacket("udp", ":9023")
    if err != nil {
        log.Fatalln("Failed to listen:", err)
    }
    cipher, err := ss.NewCipher("aes-256-cfb", "lixin93")
    if err != nil {
        log.Fatalln("Failed to create cipher:", err)
    }
    secureconn := ss.NewUDPConn(ln.(*net.UDPConn), cipher.Copy())
    goon := make(chan struct{})
    go func(){
        buf := make([]byte, 1522)
        i := 0
        for {
            n, addr, err := secureconn.ReadFrom(buf)
            mainaddr = addr
            if err != nil {
                log.Fatalln("Failed to read:", err)
            }
            if i == 0 {
                close(goon)
                i = 1
				log.Println(i)
            }
            ifce.Write(buf[:n])
        }
    }()
    <-goon
    buf := make([]byte, 1522)
    for {
        n, _ := ifce.Read(buf)
        go secureconn.WriteTo(buf[:n], mainaddr)
    }
}
