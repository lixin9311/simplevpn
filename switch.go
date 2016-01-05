package main

import (
	"github.com/lixin9311/simplevpn/tap"
	"io"
	"log"
	"sync"
)

const (
	MaxPacketSize = 1600
)

var (
	hub           = &Hub{Clients: map[tap.HwAddr]*Client{}, input: ether_buffer}
	BoradcastAddr = tap.HwAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	ether_buffer  = make(chan []byte, 4000)
)

type Hub struct {
	Clients map[tap.HwAddr]*Client
	input   <-chan []byte
	sync.Mutex
}

func (h *Hub) Init() {
	go func() {
		var dst tap.HwAddr
		//var src HwAddr
		for buf := range h.input {
			copy(dst[:], buf[:6])
			//copy(src[:], buf[6:12])
			if dst == BoradcastAddr {
				h.Boradcast(buf)
			} else {
				h.Unicast(buf, dst)
			}
		}
	}()
}

func (h *Hub) Connect(client *Client) {
	h.Lock()
	defer h.Unlock()
	log.Printf("Client with MacAddr %s connected.\n", client.MacAddr)
	h.Clients[client.MacAddr] = client
}

func (h *Hub) Disonnect(client *Client) {
	h.Lock()
	defer h.Unlock()
	log.Printf("Client with MacAddr %s disconnected.\n", client.MacAddr)
	delete(h.Clients, client.MacAddr)
}

func (h *Hub) Boradcast(data []byte) {
	h.Lock()
	defer h.Unlock()
	for _, v := range h.Clients {
		go v.Write(data)
	}
}

func (h *Hub) Unicast(data []byte, addr tap.HwAddr) (n int, err error) {
	h.Lock()
	client, ok := h.Clients[addr]
	h.Unlock()
	if !ok {
		// client do not exist
		// need a new err
		return 0, nil
	}
	n, err = client.Write(data)
	return
}

type Client struct {
	MacAddr tap.HwAddr
	Conn    io.ReadWriter
}

func (c *Client) Init() {
	go func() {
		defer hub.Disonnect(c)
		for {
			buf := make([]byte, MaxPacketSize)
			n, err := c.Conn.Read(buf)
			if err != nil {
				log.Printf("Err when read from client[%s]:%v\n", c.MacAddr, err)
				break
			}
			ether_buffer <- buf[:n]
		}
	}()
}

func (c *Client) Write(data []byte) (n int, err error) {
	n, err = c.Conn.Write(data)
	return
}

func (c *Client) Read(data []byte) (n int, err error) {
	n, err = c.Conn.Read(data)
	return
}
