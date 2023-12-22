/*
Copyright © 2023 Comcast Cable Communications Management, LLC
*/
package main

import "github.com/Comcast/Buildenv-Tool/cmd"

var version string = "development"

func main() {
	cmd.Execute(version)
}
