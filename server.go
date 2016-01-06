package main

import (
	"fmt"
	ss "github.com/lixin9311/simplevpn/securesocket"
	"github.com/lixin9311/simplevpn/tap"
	"log"
	"net"
)

func runServer(conf *Config) error {
	hub.Init()
	ifce, err := tap.NewTAP()
	if err != nil {
		log.Println("Failed to create tap interface:", err)
		return err
	}
	ip, ip_mask, err := net.ParseCIDR("192.168.1.1/24")
	if err != nil {
		return err
	}
	defer ifce.Close()
	ip_mask.IP = ip
	err = ifce.SetIP(ip_mask)
	if err != nil {
		return err
	}
	local := &Client{MacAddr: ifce.MacAddr(), Conn: ifce}
	local.Init()
	hub.Connect(local)
	if *udp {
		return serveUDP(conf, ifce.MacAddr())
	}
	return serve(conf, ifce.MacAddr())
}

func serveUDP(conf *Config, localHWAddr tap.HwAddr) error {
	cipher, err := ss.NewCipher(conf.User.Method, conf.User.Password)
	if err != nil {
		return err
	}
	test := net.ParseIP(conf.Server.Ip)
	v6 := false
	if test.To4() == nil {
		v6 = true
	}
	var addr string
	if v6 {
		addr = fmt.Sprintf("[%s]:%d", conf.Server.Ip, conf.Server.Port)
	} else {
		addr = fmt.Sprintf("%s:%d", conf.Server.Ip, conf.Server.Port)
	}

	ln, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	var listener net.PacketConn
	if *enc {
		listener = ln
	} else {
		listener = ss.NewPacketConn(ln, cipher.Copy())
	}
	defer listener.Close()
	log.Printf("Server inited, listen on : %s:%d.\n", conf.Server.Ip, conf.Server.Port)
	for {
		buf := make([]byte, MaxPacketSize)
		n, addr, err := listener.ReadFrom(buf)
		if err != nil {
			log.Println("[server]: Failed to read:", err)
		}
		if v, ok := hub.Packet_clients[addr]; !ok {
			auth := new(Auth)
			var mac tap.HwAddr
			err = auth.Unmarshal(buf[:n])
			if err != nil {
				log.Println("[Read]: Failed to Unmarshal data:", err)
			}
			if auth.Type != Auth_Hello {
				log.Println("[Read]: Unexpected message type.")
			}
			copy(mac[:], auth.MacAddr[:6])
			response := new(Auth)
			response.Type = Auth_Welcome
			response.IP = "192.168.1.101/24"
			response.DNS = "8.8.8.8"
			response.MTU = int32(1500)
			response.GateWay = "192.168.1.1"
			response.MacAddr = localHWAddr[:]
			data, err := response.Marshal()
			if err != nil {
				log.Println("Faield to encode config:", err)
			}
			client := &Client{MacAddr: mac, PacketConn: listener, is_packet: true, input: make(chan []byte, 8), remoteAddr: addr}
			client.Write(data)
			hub.Connect(client)
		} else {
			v.input <- buf[:n]
		}
	}
}

func serve(conf *Config, localHWAddr tap.HwAddr) error {
	cipher, err := ss.NewCipher(conf.User.Method, conf.User.Password)
	if err != nil {
		return err
	}
	test := net.ParseIP(conf.Server.Ip)
	v6 := false
	if test.To4() == nil {
		v6 = true
	}
	var addr string
	if v6 {
		addr = fmt.Sprintf("[%s]:%d", conf.Server.Ip, conf.Server.Port)
	} else {
		addr = fmt.Sprintf("%s:%d", conf.Server.Ip, conf.Server.Port)
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Printf("Server inited, listen on : %s:%d.\n", conf.Server.Ip, conf.Server.Port)
	// listener handler
	{
		for {
			conn, err := listener.Accept()
			if err != nil {
				return err
			}
			var c net.Conn
			if *enc {
				c = ss.NewPacketStreamConn(conn)
			} else {
				secureconn := ss.NewConn(conn, cipher.Copy())
				c = ss.NewPacketStreamConn(secureconn)
			}
			// conn handle
			go func(conn net.Conn) {
				buf := make([]byte, MaxPacketSize)
				auth := new(Auth)
				response := new(Auth)
				var mac tap.HwAddr
				// err happens, close the connection.
				defer func() {
					if err := recover(); err != nil {
						conn.Close()
					}
				}()
				{
					n, err := conn.Read(buf)
					if err != nil {
						panic(fmt.Sprintln("[Read]: Failed to read: ", err))
					}
					err = auth.Unmarshal(buf[:n])
					if err != nil {
						panic(fmt.Sprintln("[Read]: Failed to Unmarshal data:", err))
					}
					if auth.Type != Auth_Hello {
						panic(fmt.Sprintln("[Read]: Unexpected message type."))
					}
					copy(mac[:], auth.MacAddr[:6])
					response.Type = Auth_Welcome
					response.IP = "192.168.1.101/24"
					response.DNS = "8.8.8.8"
					response.MTU = int32(1500)
					response.GateWay = "192.168.1.1"
					response.MacAddr = localHWAddr[:]
					data, err := response.Marshal()
					if err != nil {
						panic(fmt.Sprintln("[Read]: Failed to marshal response: ", err))
					}
					client := &Client{MacAddr: mac, Conn: conn}
					client.Write(data)
					hub.Connect(client)
					client.Init()
				}
				return
			}(c)
		}
	}
	return nil
}
