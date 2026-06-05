//go:build !windows

package main

import "os/exec"

func setSysProcAttr(cmd *exec.Cmd) {
	// No-op for non-Windows platforms
}
