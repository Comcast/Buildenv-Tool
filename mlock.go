// +build !windows

package main

import (
	"fmt"
	"syscall"

	"github.com/urfave/cli"
	"golang.org/x/sys/unix"
)

func enableMlock(mlockBool bool) error {

	if mlockBool {
		fmt.Printf("mlock bool is: %t \n", mlockBool)
		mlockError := unix.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE)
		if mlockError != nil {
			return cli.NewExitError(fmt.Sprintf("mlock error: %s", mlockError), 1)
		}
	}
	return nil
}
