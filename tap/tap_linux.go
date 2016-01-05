// +build linux
package tap

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

var (
	IfceHwAddrNotFound = errors.New("Failed to find the hardware addr of interface.")
)

const (
	cIFF_TUN   = 0x0001
	cIFF_TAP   = 0x0002
	cIFF_NO_PI = 0x1000
)

type ifReq struct {
	Name  [0x10]byte
	Flags uint16
	pad   [0x28 - 0x10 - 2]byte
}

func createInterface(fd uintptr, ifName string, flags uint16) (createdIFName string, err error) {
	var req ifReq
	req.Flags = flags
	copy(req.Name[:], ifName)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		err = errno
		return
	}
	createdIFName = strings.Trim(string(req.Name[:]), "\x00")
	return
}

// NewTAP creates a new tap interface.
// Windows version behaves a little bit differently to Linux version.
func newTAP() (ifce *Interface, err error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	name, err := createInterface(file.Fd(), "", cIFF_TAP|cIFF_NO_PI)
	if err != nil {
		return nil, err
	}
	ifce = &Interface{tap: true, file: file, name: name}
	// find the mac address of interface.
	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, v := range ifaces {
		if v.Name == name {
			copy(ifce.mac[:6], v.HardwareAddr[:6])
			return ifce, ifce.up()
		}
	}
	err = IfceHwAddrNotFound
	return
}

func (ifce *Interface) up() (err error) {
	sargs := fmt.Sprintf("link set dev %s up mtu 1400", ifce.name)
	args := strings.Split(sargs, " ")
	cmd := exec.Command("ip", args...)
	err = cmd.Run()
	return
}

func (ifce *Interface) setIP(ip_mask *net.IPNet) (err error) {
	sargs := fmt.Sprintf("addr add %s dev %s", ip_mask, ifce.name)
	args := strings.Split(sargs, " ")
	cmd := exec.Command("ip", args...)
	err = cmd.Run()
	return
}

func addRoute(ip net.IP, ip_mask *net.IPNet, ifce string) (err error) {
	sargs := fmt.Sprintf("route add %s via %s dev %s", ip_mask, ip, ifce)
	args := strings.Split(sargs, " ")
	cmd := exec.Command("ip", args...)
	err = cmd.Run()
	return
}

func (ifce *Interface) addRoute(ip net.IP, ip_mask *net.IPNet) (err error) {
	return addRoute(ip, ip_mask, ifce.name)
}

func delRoute(ip net.IP, ip_mask *net.IPNet, ifce string) (err error) {
	sargs := fmt.Sprintf("route del %s via %s dev %s", ip_mask, ip, ifce)
	args := strings.Split(sargs, " ")
	cmd := exec.Command("ip", args...)
	err = cmd.Run()
	return
}

func (ifce *Interface) delRoute(ip net.IP, ip_mask *net.IPNet) (err error) {
	return delRoute(ip, ip_mask, ifce.name)
}
