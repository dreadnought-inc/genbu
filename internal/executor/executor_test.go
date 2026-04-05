package executor

import (
	"testing"
)

func TestExec_noCommand(t *testing.T) {
	err := Exec(nil, nil)
	if err == nil {
		t.Fatal("expected error for empty args")
	}
}

func TestExec_commandNotFound(t *testing.T) {
	err := Exec([]string{"nonexistent-command-12345"}, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}
}
