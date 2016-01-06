package tap

import (
	"fmt"
	"net"
	"os"
	"sync"
)

// 48 bits Mac addr
type HwAddr [6]byte

func (h HwAddr) String() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", h[0], h[1], h[2], h[3], h[4], h[5])
}

// Interface is the abstract class of an network interface.
type Interface struct {
	rlock sync.Mutex
	wlock sync.Mutex
	tap   bool
	file  *os.File
	name  string
	mac   HwAddr
}

// Create a new tap device.
// Windows version behaves a little bit differently to Linux version.
func NewTAP() (ifce *Interface, err error) {
	return newTAP()
}

// Returns true if ifce is a TUN interface, otherwise returns false;
func (ifce *Interface) IsTUN() bool {
	return !ifce.tap
}

// Returns true if ifce is a TAP interface, otherwise returns false;
func (ifce *Interface) IsTAP() bool {
	return ifce.tap
}

// Returns the interface name of ifce, e.g. tun0, tap1, etc..
func (ifce *Interface) Name() string {
	return ifce.name
}

// Implement io.Writer interface.
func (ifce *Interface) Write(p []byte) (int, error) {
	ifce.wlock.Lock()
	defer ifce.wlock.Unlock()
	return ifce.file.Write(p)
}

// Implement io.Reader interface.
func (ifce *Interface) Read(p []byte) (int, error) {
	ifce.rlock.Lock()
	defer ifce.rlock.Unlock()
	return ifce.file.Read(p)
}

// Close the interface.
func (ifce *Interface) Close() error {
	return ifce.file.Close()
}

// Mac address of the interface.
func (ifce *Interface) MacAddr() HwAddr {
	return ifce.mac
}

// Set ip address of the interface.
func (ifce *Interface) SetIP(ip_mask *net.IPNet) error {
	return ifce.setIP(ip_mask)
}

func (ifce *Interface) AddRoute(ip net.IP, ip_mask *net.IPNet) error {
	return ifce.addRoute(ip, ip_mask)
}

func (ifce *Interface) DelRoute(ip net.IP, ip_mask *net.IPNet) error {
	return ifce.delRoute(ip, ip_mask)
}
