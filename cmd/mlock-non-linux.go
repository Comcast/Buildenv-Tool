//go:build !linux
// +build !linux

package cmd

func EnableMlock() error {
	// NOTE: Not required for windows
	return nil
}
