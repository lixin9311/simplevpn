package tap

import (
	"net"
)

var bypassed_gateway_ip []net.IP
var bypassed_dst []*net.IPNet
var bypassed_dev []string

func Bypass(ip_mask *net.IPNet) error {
	ip, dev, err := find_gateway(ip_mask)
	if err != nil {
		return err
	}
	err = addRoute(ip, ip_mask, dev)
	if err != nil {
		return err
	}
	bypassed_gateway_ip = append(bypassed_gateway_ip, ip)
	backup_ip_mask := *ip_mask
	bypassed_dst = append(bypassed_dst, &backup_ip_mask)
	bypassed_dev = append(bypassed_dev, dev)
	return nil
}

func Unbypass() {
	for n := len(bypassed_dev); n > 0; n-- {
		delRoute(bypassed_gateway_ip[n-1], bypassed_dst[n-1], bypassed_dev[n-1])
	}
	bypassed_dev = nil
	bypassed_dst = nil
	bypassed_gateway_ip = nil
}

func AddRoute(ip net.IP, ip_mask *net.IPNet, ifce string) error {
	return addRoute(ip, ip_mask, ifce)
}

func DelRoute(ip net.IP, ip_mask *net.IPNet, ifce string) error {
	return delRoute(ip, ip_mask, ifce)
}
