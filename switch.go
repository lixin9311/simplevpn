package main

import (
	"fmt"
	"github.com/lixin9311/simplevpn/tap"
	"io"
	"log"
	"net"
	"sync"
)

const (
	MaxPacketSize = 1600
)

var (
	hub           = &Hub{Clients: map[tap.HwAddr]*Client{}, input: ether_buffer}
	BroadcastAddr = tap.HwAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	ether_buffer  = make(chan []byte, 4000)
)

type Hub struct {
	Clients        map[tap.HwAddr]*Client
	Packet_clients map[net.Addr]*Client
	input          chan []byte
	sync.Mutex
}

func (h *Hub) Init() {
	go func() {
		var dst tap.HwAddr
		var src tap.HwAddr
		for buf := range h.input {
			copy(dst[:], buf[:6])
			copy(src[:], buf[6:12])
			if dst == BroadcastAddr {
				go h.Broadcast(buf, src)
			} else {
				go h.Unicast(buf, dst)
			}
		}
	}()
}

func (h *Hub) Connect(client *Client) {
	h.Lock()
	defer h.Unlock()
	log.Printf("Client with MacAddr %s connected.\n", client.MacAddr)
	h.Clients[client.MacAddr] = client
	if client.is_packet {
		h.Packet_clients[client.remoteAddr] = client
	}
}

func (h *Hub) Disonnect(client *Client) {
	h.Lock()
	defer h.Unlock()
	log.Printf("Client with MacAddr %s disconnected.\n", client.MacAddr)
	delete(h.Clients, client.MacAddr)
	if client.is_packet {
		delete(h.Packet_clients, client.remoteAddr)
	}
}

func (h *Hub) Broadcast(data []byte, src tap.HwAddr) {
	h.Lock()
	defer h.Unlock()
	for k, v := range h.Clients {
		if k == src {
			continue
		}
		go v.Write(data)
	}
}

func (h *Hub) Unicast(data []byte, addr tap.HwAddr) (n int, err error) {
	h.Lock()
	client_u, ok_u := h.Clients[addr]
	client_s, ok_s := h.Clients[BroadcastAddr]
	h.Unlock()
	if ok_u {
		n, err = client_u.Write(data)
	}
	if ok_s {
		go client_s.Write(data)
	}
	return
}

type Client struct {
	MacAddr    tap.HwAddr
	Conn       io.ReadWriter
	is_packet  bool
	PacketConn net.PacketConn
	input      chan []byte
	remoteAddr net.Addr
}

func (c *Client) Init() {
	go func() {
		defer hub.Disonnect(c)
		for {
			buf := make([]byte, MaxPacketSize)
			n, err := c.Read(buf)
			if err != nil {
				log.Printf("Err when read from client[%s]:%v\n", c.MacAddr, err)
				break
			}
			ether_buffer <- buf[:n]
		}
	}()
}

func (c *Client) Run() error {
	defer hub.Disonnect(c)
	for {
		buf := make([]byte, MaxPacketSize)
		n, err := c.Read(buf)
		if err != nil {
			log.Printf("Err when read from client[%s]:%v\n", c.MacAddr, err)
			return err
		}
		ether_buffer <- buf[:n]
	}
}

func (c *Client) Write(data []byte) (n int, err error) {
	if c.is_packet {
		n, err = c.PacketConn.WriteTo(data, c.remoteAddr)
	} else {
		n, err = c.Conn.Write(data)
	}
	return
}

func (c *Client) Read(data []byte) (n int, err error) {
	if c.is_packet {
		buf := <-c.input
		if len(data) < len(buf) {
			log.Println("Buffer too short:", len(buf))
			err = fmt.Errorf("Buffer too short")
			return
		}
		copy(data, buf)
		n = len(buf)
	} else {
		n, err = c.Conn.Read(data)
	}
	return
}
