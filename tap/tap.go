package tap

import (
	"os"
)

// Interface is the abstract class of an network interface.
type Interface struct {
	tap  bool
	file *os.File
	name string
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
	return ifce.file.Write(p)
}

// Implement io.Reader interface.
func (ifce *Interface) Read(p []byte) (int, error) {
	return ifce.file.Read(p)
}

// Close the interface.
func (ifce *Interface) Close() error {
	return ifce.file.Close()
}
