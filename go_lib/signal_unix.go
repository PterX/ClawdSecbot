//go:build !windows

package main

import (
	"os/signal"
	"syscall"
)

func resetSignals() {
	signal.Reset(syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)
}
