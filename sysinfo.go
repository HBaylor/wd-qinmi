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
		// wmic 在 Win11 24H2 起已被移除，改用 CIM（Get-CimInstance）
		out, err := exec.Command("powershell", "-NoProfile", "-Command",
			"(Get-CimInstance Win32_BaseBoard).SerialNumber").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	default: // linux
		out, err := exec.Command("cat", "/sys/class/dmi/id/board_serial").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
}
