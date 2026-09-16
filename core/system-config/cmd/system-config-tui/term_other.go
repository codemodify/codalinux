//go:build !linux

package main

import "fmt"

func isTTY(int) bool         { return false }
func winsize(int) (int, int) { return 80, 24 }
func makeRaw(int) (func(), error) {
	return func() {}, fmt.Errorf("raw terminal not supported on this OS")
}
