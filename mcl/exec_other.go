//go:build !windows

package mcl

import "os/exec"

// hideWindow starts a program without a console window. Only Windows shows a
// window for a program that was started from a program without one.
func hideWindow(command *exec.Cmd) {}
