// +build linux
package tap

import (
	"net"
	"os"
	"strings"
	"syscall"
	"unsafe"
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
	ifce.mac, err = ifce.macaddr()
	return
}

func (ifce *Interface) macaddr() ([]byte, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil
	}
	for _, v := range ifaces {
		if v.Name == ifce.name {
			return []byte(v.HardwareAddr), nil
		}
	}
	return nil, nil
}
