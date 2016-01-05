package main

import (
	"fmt"
	ss "github.com/lixin9311/simplevpn/securesocket"
	"github.com/lixin9311/simplevpn/tap"
	"io"
	"log"
	"net"
)

func runClient(conf *Config) error {
	cipher, err := ss.NewCipher(conf.User.Method, conf.User.Password)
	if err != nil {
		return err
	}
	ifce, err := tap.NewTAP()
	if err != nil {
		log.Println("Failed to create TAP device:", err)
		return err
	}
	defer ifce.Close()
	ip, ip_mask, err := net.ParseCIDR(conf.Server.Ip + "/32")
	if err != nil {
		log.Println("Failed to parse ip:", err)
		return err
	}
	ip_mask.IP = ip
	err = tap.Bypass(ip_mask)
	if err != nil {
		log.Println("[Client]: Failed to bypass server address from route:", err)
		return err
	}
	defer tap.Unbypass()
	// reg with server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", conf.Server.Ip, conf.Server.Port))
	if err != nil {
		return err
	}
	secureconn := ss.NewConn(conn, cipher.Copy())
	c := ss.NewPacketStreamConn(secureconn)
	defer c.Close()
	auth := new(Auth)
	auth.Type = Auth_Hello
	mac := ifce.MacAddr()
	auth.MacAddr = mac[:]
	data, err := auth.Marshal()
	if err != nil {
		log.Println("[Client]: Failed to marshal data:", err)
		return err
	}
	_, err = c.Write(data)
	if err != nil {
		log.Println("[Client]: Failed to write socket: ", err)
		return err
	}
	buf := make([]byte, 2048)
	n, err := c.Read(buf)
	if err != nil {
		log.Println("[Client]: Failed to recieve config: ", err)
		return err
	}
	err = auth.Unmarshal(buf[:n])
	if err != nil {
		log.Println("[Client]: Failed to decode recieved config: ", err)
		return err
	}
	if auth.Type != Auth_Welcome {
		return fmt.Errorf("[Client]: Unexpected response type: %s.", Auth_MessageType_name[int32(auth.Type)])
	}
	ip, ip_mask, err = net.ParseCIDR(auth.IP)
	if err != nil {
		log.Println("[Client]: Failed to parse CIDR from response:", err)
		return err
	}
	ip_mask.IP = ip
	err = ifce.SetIP(ip_mask)
	if err != nil {
		log.Println("Failed to set IP address:", err)
		return err
	}
	go PipeThenClose(c, ifce)
	PipeThenClose(ifce, c)
	return nil
}

func PipeThenClose(src, dst io.ReadWriteCloser) {
	defer dst.Close()
	buf := make([]byte, 1522)
	for {
		n, err := src.Read(buf)
		// read may return EOF with n > 0
		// should always process n > 0 bytes before handling error
		if n > 0 {
			// Note: avoid overwrite err returned by Read.
			if _, err := dst.Write(buf[0:n]); err != nil {
				log.Println("write:", err)
				break
			}
		}
		if err != nil {
			// Always "use of closed network connection", but no easy way to
			// identify this specific error. So just leave the error along for now.
			// More info here: https://code.google.com/p/go/issues/detail?id=4373
			/*
				if bool(Debug) && err != io.EOF {
					Debug.Println("read:", err)
				}
			*/
			break
		}
	}
}
