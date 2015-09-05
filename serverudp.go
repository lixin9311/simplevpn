package main

import (
	"log"
    "net"
	ss "github.com/lixin9311/simplevpn/securesocket"
)

func main() {
    ln, err := net.ListenPacket("udp", ":9023")
    if err != nil {
        log.Fatalln("Failed to listen:", err)
    }
    cipher, err := ss.NewCipher("aes-256-cfb", "lixin93")
    if err != nil {
        log.Fatalln("Failed to create cipher:", err)
    }
    secureconn := ss.NewUDPConn(ln.(*net.UDPConn), cipher.Copy())
	buf := make([]byte, 1522)
	for {
		n, src, err := secureconn.ReadFrom(buf)
		log.Println("Read:", buf[:n])
		if err != nil {
			log.Fatalln("Failed to read:", err)
		}
		go secureconn.WriteTo(buf[:n], src)
	}
}
