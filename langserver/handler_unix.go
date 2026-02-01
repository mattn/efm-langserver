//go:build unix

// These functions only work on unix-like operating systems. (linux, darwin, ???)

package langserver

import (
	"context"
	"os/exec"
	"syscall"
)

// killableComand configures a command so that it *and* all of its children will be killed when
// 'ctx' is cancelled.
// See: https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
func killableCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// By default, exec.CommandContext() sets Cancel() to only kill the main process.
	// (In this case, `sh`.)
	// But we want to kill that process and any processes it may have spawned.

	// Create a new process group when we spawn this process:
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Update the Cancel function to kill the whole process group:
	cmd.Cancel = func() error {
		// The new pgid is the same as the parent process:
		pgid := cmd.Process.Pid
		// Use -pgid to show we mean to kill a group ID:
		return syscall.Kill(-pgid, syscall.SIGKILL)
	}

	return cmd
}
