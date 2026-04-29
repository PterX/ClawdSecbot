//go:build windows

package cmdutil

import (
	"os/exec"
	"testing"
)

func TestPrepareHidesWindowsConsole(t *testing.T) {
	cmd := exec.Command("botsec-test")

	Prepare(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("Expected SysProcAttr to be initialized")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("Expected child window to be hidden")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatalf("Expected createNoWindow flag, got %#x", cmd.SysProcAttr.CreationFlags)
	}
}

func TestPrepareHandlesNilCommand(t *testing.T) {
	Prepare(nil)
}
