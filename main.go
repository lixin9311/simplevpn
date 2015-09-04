package main

import (
	"fmt"
	"github.com/lixin9311/simplevpn/tap"
	"github.com/songgao/water/waterutil"
)

const BUFFERSIZE = 1522

func main() {
	ifce, err := tap.NewTAP()
	fmt.Printf("%v, %v\n\n", err, ifce)
	buffer := make([]byte, BUFFERSIZE)
	for {
		_, err = ifce.Read(buffer)
		if err != nil {
			break
		}
		ethertype := waterutil.MACEthertype(buffer)
		if ethertype == waterutil.IPv4 {
			packet := waterutil.MACPayload(buffer)
			if waterutil.IsIPv4(packet) {
				fmt.Printf("Source:      %v [%v]\n", waterutil.MACSource(buffer), waterutil.IPv4Source(packet))
				fmt.Printf("Destination: %v [%v]\n", waterutil.MACDestination(buffer), waterutil.IPv4Destination(packet))
				fmt.Printf("Protocol:    %v\n\n", waterutil.IPv4Protocol(packet))
			}
		}
	}
}
