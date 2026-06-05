//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

const CREATE_NO_WINDOW = 0x08000000

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: CREATE_NO_WINDOW,
	}
}
