package main

import (
	ss "./securesocket"
	"./tap"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/lixin9311/icli-go"
	"io"
	"log"
	"net"
	"os"
)

var (
	tinyErr  = errors.New("Tiny error.")
	sysFault = errors.New("System Fault.")
	ifce     *tap.Interface
	listener net.Listener
	cipher   *ss.Cipher
)

func listen(args ...string) error {
	var err error
	listener, err = net.Listen("tcp", ":9999")
	if err != nil {
		fmt.Println("Failed to listen port 9999:", err)
		return err
	}
	cipher, err = ss.NewCipher("aes-256-cfb", "passwd")
	if err != nil {
		fmt.Println("Failed to create cipher:", err)
	}
	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Failed to accept new conn:", err)
				break
			}
			secureconn := ss.NewTCPConn(conn, cipher.Copy())
			go func(conn net.Conn) {
				var size uint
				buf := make([]byte, 1500)
				defer conn.Close()
				for {
					binary.Read(conn, binary.BigEndian, &size)
					fmt.Printf("[Read]: incomming packet size: %d\n", size)
					n, err := io.ReadAtLeast(conn, buf, int(size))
					if err != nil {
						fmt.Println("[Read]: Failed to read: ", err)
						break
					}
					fmt.Println("[Read]: incoming content: ", string(buf[:n]))
				}
			}(secureconn)

		}
	}()
}

func newtap(args ...string) error {
	var err error
	ifce, err = tap.NewTAP()
	if err != nil {
		return err
	}
	fmt.Println("Created TAP name:", ifce.Name())
	fmt.Println("Created TAP mac:", ifce.MacAddr())
	//buf := make([]byte, 512)
	return nil
}

func allifs(args ...string) error {
	ifs, _ := net.Interfaces()
	for _, inter := range ifs {
		fmt.Println([]byte(inter.HardwareAddr))
		fmt.Println(inter)
	}
	return nil
}

func exit(args ...string) error {
	return icli.ExitIcli
}

// should also process nil error
func errorhandler(e error) error {
	return nil
}
func main() {
	log.SetOutput(os.Stdout)
	icli.AddCmd([]icli.CommandOption{
		{"new", "test", newtap},
		{"list", "test", allifs},
		{"exit", "exit", exit},
	})
	// utf-8 safe
	icli.SetPromt("输入 input >")
	icli.Start(errorhandler)
}
