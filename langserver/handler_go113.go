// +build go1.13

package langserver

import (
	"os/exec"
)

func succeeded(err error) bool {
	exitErr, ok := err.(*exec.ExitError)
	// When the context is canceled, the process is killed,
	// and the exit code is -1
	return ok && exitErr.ExitCode() < 0
}
