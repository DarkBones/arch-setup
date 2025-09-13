package system

import (
	"bufio"
	"os"
	"runtime"
	"strings"
)

type OSInfo struct {
	Family string
	Distro string
}

func CurrentOSInfo() OSInfo {
	fam := runtime.GOOS
	info := OSInfo{Family: fam}

	if fam == "linux" {
		if id := linuxDistroID(); id != "" {
			info.Distro = id
		}
	}
	if fam == "darwin" {
		info.Distro = "macos"
	}
	return info
}

func linuxDistroID() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "ID=") {
			continue
		}

		// ID=arch OR ID="arch"
		val := strings.TrimPrefix(line, "ID=")
		val = strings.Trim(val, `"'`)
		return strings.ToLower(strings.TrimSpace(val))
	}
	return ""
}
