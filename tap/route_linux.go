// +build linux
package tap

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

func find_gateway(ip_mask *net.IPNet) (ip net.IP, dev string, err error) {
	sargs := fmt.Sprintf("route get %s", ip_mask)
	args := strings.Split(sargs, " ")
	cmd := exec.Command("ip", args...)
	out, err := cmd.Output()
	if err != nil {
		return
	}
	buf := bytes.NewBuffer(out)
	firstline, _ := buf.ReadString('\n')
	fields := strings.Fields(firstline)
	if len(fields) < 5 {
		err = fmt.Errorf("Invalid result:%s", firstline)
		return
	}
	ip = net.ParseIP(fields[2])
	dev = fields[4]
	return
}
