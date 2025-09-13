package main

import (
	"os"
	"os/exec"
)

type PrivilegeChecker interface {
	Check() error
}

type SudoChecker struct{}

func (SudoChecker) Check() error {
	if exec.Command("sudo", "-n", "true").Run() == nil {
		return nil
	}
	cmd := exec.Command("sudo", "-v")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
