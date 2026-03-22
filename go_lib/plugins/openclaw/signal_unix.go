//go:build !windows

package openclaw

import (
	"os/signal"
	"syscall"
)

func resetPluginSignals() {
	signal.Reset(syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)
}
