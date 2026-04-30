package cmdutil

import "testing"

func TestCommandPreparesNameAndArgs(t *testing.T) {
	cmd := Command("botsec-test", "one", "two")

	if cmd.Path != "botsec-test" {
		t.Fatalf("Expected command path botsec-test, got %q", cmd.Path)
	}
	if len(cmd.Args) != 3 || cmd.Args[0] != "botsec-test" || cmd.Args[1] != "one" || cmd.Args[2] != "two" {
		t.Fatalf("Expected command args to be preserved, got %#v", cmd.Args)
	}
}

func TestBackgroundCommandSilencesStandardStreams(t *testing.T) {
	cmd := BackgroundCommand("botsec-test")

	if cmd.Stdin == nil || cmd.Stdout == nil || cmd.Stderr == nil {
		t.Fatalf("Expected background command stdio to be redirected, stdin=%v stdout=%v stderr=%v", cmd.Stdin, cmd.Stdout, cmd.Stderr)
	}
}

func TestSilenceHandlesNilCommand(t *testing.T) {
	Silence(nil)
}
