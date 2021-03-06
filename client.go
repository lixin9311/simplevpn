package main

import (
	"fmt"
	ss "github.com/lixin9311/simplevpn/securesocket"
	"github.com/lixin9311/simplevpn/tap"
	"io"
	"log"
	"net"
)

func runUDP(conf *Config) error {
	hub.Init()
	test := net.ParseIP(conf.Server.Ip)
	v6 := false
	if test.To4() == nil {
		v6 = true
	}
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
	local := &Client{MacAddr: ifce.MacAddr(), Conn: ifce}
	local.Init()
	hub.Connect(local)
	if !v6 {
		ip, ip_mask, err := net.ParseCIDR(conf.Server.Ip + "/32")
		if err != nil {
			log.Println("Failed to parse ip:", err)
			return err
		}
		ip_mask.IP = ip
		err = tap.Bypass(ip_mask)
		if err != nil {
			log.Println("[Client]: Failed to bypass server address from route, please manually fix that:", err)
		}
		defer tap.Unbypass()
	}
	// reg with server
	var dst string
	if v6 {
		dst = fmt.Sprintf("[%s]:%d", conf.Server.Ip, conf.Server.Port)
	} else {
		dst = fmt.Sprintf("%s:%d", conf.Server.Ip, conf.Server.Port)
	}
	remote, err := net.ResolveUDPAddr("udp", dst)
	if err != nil {
		log.Println("Faild to resolve addr:", err)
	}
	var ln net.PacketConn
	if v6 {
		ln, err = net.ListenPacket("udp", "::9911")
	} else {
		ln, err = net.ListenPacket("udp", ":9911")
	}
	listener := ss.NewPacketConn(ln, cipher.Copy())
	auth := new(Auth)
	auth.Type = Auth_Hello
	mac := ifce.MacAddr()
	auth.MacAddr = mac[:]
	data, err := auth.Marshal()
	if err != nil {
		log.Println("[Client]: Failed to marshal data:", err)
		return err
	}
	_, err = listener.WriteTo(data, remote)
	if err != nil {
		log.Println("[Client]: Failed to write socket: ", err)
		return err
	}
	buf := make([]byte, 2048)
reread:
	n, addr, err := listener.ReadFrom(buf)
	if addr != remote {
		log.Println("Address mismatch, reread.")
		goto reread
	}
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
	ip, ip_mask, err := net.ParseCIDR(auth.IP)
	if err != nil {
		log.Println("[Client]: Failed to parse CIDR from response:", err)
		return err
	}
	ip_mask.IP = ip
	err = ifce.SetIP(ip_mask)
	if err != nil {
		log.Println("Failed to set IP address:", err)
		log.Println("Maybe you can manually fix that, so go on.")
		//return err
	}
	ip, ip_mask, err = net.ParseCIDR("0.0.0.0/1")
	if err != nil {
		log.Println("Failed to parse address:", err)
	}
	ip_mask.IP = ip
	ip = net.ParseIP(auth.GateWay)
	err = ifce.AddRoute(ip, ip_mask)
	if err != nil {
		log.Println("Failed to set default route, please manually fix that:", err)
	}
	ip, ip_mask, err = net.ParseCIDR("128.0.0.0/1")
	if err != nil {
		log.Println("Failed to parse address:", err)
	}
	ip_mask.IP = ip
	ip = net.ParseIP(auth.GateWay)
	err = ifce.AddRoute(ip, ip_mask)
	if err != nil {
		log.Println("Failed to set default route, please manually fix that", err)
	}
	defer func() {
		ip, ip_mask, _ = net.ParseCIDR("0.0.0.0/1")
		ip_mask.IP = ip
		ip = net.ParseIP(auth.GateWay)
		ifce.DelRoute(ip, ip_mask)
		ip, ip_mask, _ = net.ParseCIDR("128.0.0.0/1")
		ip_mask.IP = ip
		ip = net.ParseIP(auth.GateWay)
		ifce.DelRoute(ip, ip_mask)
	}()
	client := &Client{MacAddr: BroadcastAddr, PacketConn: listener, is_packet: true, input: make(chan []byte, 8), remoteAddr: remote}
	hub.Connect(client)
	client.Init()
	for {
		buf := make([]byte, MaxPacketSize)
		n, addr, err := listener.ReadFrom(buf)
		if err != nil {
			log.Println("Failed to read:", err)
			continue
		}
		if addr != remote {
			continue
		}
		client.input <- buf[:n]
	}

}

func runClient(conf *Config) error {
	if *udp {
		return runUDP(conf)
	}
	hub.Init()
	test := net.ParseIP(conf.Server.Ip)
	v6 := false
	if test.To4() == nil {
		v6 = true
	}
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
	local := &Client{MacAddr: ifce.MacAddr(), Conn: ifce}
	local.Init()
	hub.Connect(local)
	if !v6 {
		ip, ip_mask, err := net.ParseCIDR(conf.Server.Ip + "/32")
		if err != nil {
			log.Println("Failed to parse ip:", err)
			return err
		}
		ip_mask.IP = ip
		err = tap.Bypass(ip_mask)
		if err != nil {
			log.Println("[Client]: Failed to bypass server address from route, please manually fix that:", err)
		}
		defer tap.Unbypass()
	}
	// reg with server
	var dst string
	if v6 {
		dst = fmt.Sprintf("[%s]:%d", conf.Server.Ip, conf.Server.Port)
	} else {
		dst = fmt.Sprintf("%s:%d", conf.Server.Ip, conf.Server.Port)
	}
	conn, err := net.Dial("tcp", dst)
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
	ip, ip_mask, err := net.ParseCIDR(auth.IP)
	if err != nil {
		log.Println("[Client]: Failed to parse CIDR from response:", err)
		return err
	}
	ip_mask.IP = ip
	err = ifce.SetIP(ip_mask)
	if err != nil {
		log.Println("Failed to set IP address:", err)
		log.Println("Maybe you can manually fix that, so go on.")
		//return err
	}
	ip, ip_mask, err = net.ParseCIDR("0.0.0.0/1")
	if err != nil {
		log.Println("Failed to parse address:", err)
	}
	ip_mask.IP = ip
	ip = net.ParseIP(auth.GateWay)
	err = ifce.AddRoute(ip, ip_mask)
	if err != nil {
		log.Println("Failed to set default route, please manually fix that:", err)
	}
	ip, ip_mask, err = net.ParseCIDR("128.0.0.0/1")
	if err != nil {
		log.Println("Failed to parse address:", err)
	}
	ip_mask.IP = ip
	ip = net.ParseIP(auth.GateWay)
	err = ifce.AddRoute(ip, ip_mask)
	if err != nil {
		log.Println("Failed to set default route, please manually fix that", err)
	}
	defer func() {
		ip, ip_mask, _ = net.ParseCIDR("0.0.0.0/1")
		ip_mask.IP = ip
		ip = net.ParseIP(auth.GateWay)
		ifce.DelRoute(ip, ip_mask)
		ip, ip_mask, _ = net.ParseCIDR("128.0.0.0/1")
		ip_mask.IP = ip
		ip = net.ParseIP(auth.GateWay)
		ifce.DelRoute(ip, ip_mask)
	}()
	client := &Client{MacAddr: BroadcastAddr, Conn: c}
	hub.Connect(client)
	return client.Run()
}

func PipeThenClose(src, dst io.ReadWriteCloser) {
	defer dst.Close()
	buf := make([]byte, MaxPacketSize)
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
			log.Println("read:", err)
			break
		}
	}
}
