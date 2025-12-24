//go:build !unix

package langserver

import (
	"context"
	"os/exec"
)

func killableCommand(_ context.Context, _ string) *exec.Cmd {
	panic("killableCommand() should not be called from non-unix systems")
	// TODO: There may be a Windows-equivalent implementation for the unix one,
	// but I'll leave that to a Windows user to implement and test. :)
}
