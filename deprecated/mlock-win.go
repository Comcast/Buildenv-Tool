// +build windows

package main

import (
	"fmt"

	"github.com/urfave/cli"
)

func enableMlock(mlockBool bool) error {

	if mlockBool {
		return cli.NewExitError(fmt.Sprintf("mlock not necessary for windows"), 1)
	}
	return nil
}
