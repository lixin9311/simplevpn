// +build windows
package tap

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

var default_gateway net.IP
var default_gateway_if_index string

func find_gateway(ip_mask *net.IPNet) (ip net.IP, dev string, err error) {
	if default_gateway != nil {
		return default_gateway, default_gateway_if_index, nil
	}
	sargs := "interface ip show route"
	args := strings.Split(sargs, " ")
	cmd := exec.Command("netsh", args...)
	out, err := cmd.Output()
	if err != nil {
		return
	}
	buf := bytes.NewBuffer(out)
	for {
		var line string
		line, err = buf.ReadString('\n')
		if err != nil && line == "" {
			break
		}
		if strings.Contains(line, "0.0.0.0") {
			fields := strings.Fields(line)
			if len(fields) < 6 {
				err = fmt.Errorf("Something went wrong! the length of field should not be %d!", len(fields))
				return
			}
			if fields[3] == "0.0.0.0/0" {
				default_gateway = net.ParseIP(fields[5])
				default_gateway_if_index = fields[4]
				return default_gateway, default_gateway_if_index, nil
			} else {
				err = fmt.Errorf("Something went wrong! the 4th of field should not be %s!", fields[3])
				return
			}
		}
	}
	err = fmt.Errorf("Default gatewat not found.")
	return
}
