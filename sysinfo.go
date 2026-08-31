package main

import (
	"os/exec"
	"runtime"
	"strings"
)

// getMachineID 读取电脑主板序列号作为机器标识
func getMachineID() string {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("ioreg", "-l").Output()
		if err != nil {
			return ""
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "IOPlatformSerialNumber") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					return strings.Trim(strings.TrimSpace(parts[1]), `"`)
				}
			}
		}
		return ""
	case "windows":
		out, err := exec.Command("wmic", "baseboard", "get", "SerialNumber").Output()
		if err != nil {
			return ""
		}
		lines := strings.Fields(string(out))
		if len(lines) >= 2 {
			return lines[len(lines)-1]
		}
		return ""
	default: // linux
		out, err := exec.Command("cat", "/sys/class/dmi/id/board_serial").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
}
