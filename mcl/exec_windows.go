//go:build windows

package mcl

import (
	"os/exec"
	"syscall"
)

// hideWindow starts a program without a console window, so no window flashes
// up while an installer runs.
func hideWindow(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
