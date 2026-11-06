//go:build !windows

package mcl

import (
	"os"
	"strings"
)

// osVersion returns the version of the operating system. It is matched
// against the "version" field of the rules in the version metadata.
//
// Only Linux reports a usable version; the macOS kernel version does not
// match the versions the rules expect.
func osVersion() string {
	if _, err := os.Stat("/proc/sys/kernel/osrelease"); err != nil {
		return ""
	}
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
