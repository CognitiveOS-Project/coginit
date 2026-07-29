//go:build !linux

package coginit

import "os/exec"

// execCommand creates a command without process group management on non-Linux.
func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// registerChildProcess is a no-op on non-Linux platforms.
func registerChildProcess(name string, cmd *exec.Cmd) {}

// deleteChildProcess is a no-op on non-Linux platforms.
func deleteChildProcess(pid int) {}

// supervise is a no-op on non-Linux platforms.
func supervise(name string, cmd *exec.Cmd, restartFn func() *exec.Cmd) {}
