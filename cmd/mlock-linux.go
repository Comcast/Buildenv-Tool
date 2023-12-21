//go:build linux
// +build linux

package cmd

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

func EnableMlock() error {
	mlockError := unix.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE)
	if mlockError != nil {
		return fmt.Errorf("mlock error: %w", mlockError)
	}
	return nil
}
