// +build linux
package tap

import (
	"fmt"
	"log"
	"net"
	"sync"
	"syscall"
	"unsafe"
)

type route struct {
	gateway  net.IP
	dst      *net.IPNet
	if_index int
}

func find_gateway(ip_mask *net.IPNet) (ip net.IP, dev string, err error) {
	buf, err := syscall.NetlinkRIB(syscall.RTM_GETROUTE, syscall.AF_INET)
	if err != nil {
		log.Println("Failed to open netlink:", err)
		return
	}
	msgs, err := syscall.ParseNetlinkMessage(buf)
	if err != nil {
		log.Println("Failed to parse nl msg:", err)
		return
	}
	var def route
	set := false
	var once sync.Once
loop:
	for _, m := range msgs {
		switch m.Header.Type {
		case syscall.NLMSG_DONE:
			break loop
		case syscall.RTM_NEWROUTE:
			// a route enrty
			var r route
			rtmsg := (*syscall.RtMsg)(unsafe.Pointer(&m.Data[0]))
			attrs, err := syscall.ParseNetlinkRouteAttr(&m)
			if err != nil {
				// err is shadowed
				log.Println("Failed to parse nl rtattr:", err)
				return ip, dev, err
			}
			// parse a route entry
			for _, a := range attrs {
				switch a.Attr.Type {
				case syscall.RTA_DST:
					addr := a.Value
					_, r.dst, err = net.ParseCIDR(fmt.Sprintf("%d.%d.%d.%d/%d", addr[0], addr[1], addr[2], addr[3], rtmsg.Dst_len))
					if err != nil {
						log.Println("Failed to parse ip addr:", err)
						return ip, dev, err
					}
				case syscall.RTA_GATEWAY:
					addr := a.Value
					r.gateway = net.IPv4(addr[0], addr[1], addr[2], addr[3])
				case syscall.RTA_OIF:
					r.if_index = int(a.Value[0])
				}
			}
			if r.dst == nil {
				once.Do(func() {
					def = r
					set = true
				})
			} else {
				if r.dst.Contains(ip_mask.IP) {
					ifce, err := net.InterfaceByIndex(r.if_index)
					if err != nil {
						log.Println("Failed to get interface by index:", err)
						return ip, dev, err
					}
					return r.gateway, ifce.Name, nil
				}

			}
		}
	}
	if set {
		ifce, err := net.InterfaceByIndex(def.if_index)
		if err != nil {
			log.Println("Failed to get interface by index:", err)
			return ip, dev, err
		}
		return def.gateway, ifce.Name, nil
	}
	err = fmt.Errorf("Route not found.")
	return
}
