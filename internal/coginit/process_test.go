//go:build linux

package coginit

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestExecCommand_SetsProcessGroup(t *testing.T) {
	cmd := execCommand("echo", "hello")
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should not be nil")
	}
	if !cmd.SysProcAttr.Setpgid {
		t.Fatal("Setpgid should be true")
	}
}

func TestRegisterAndDeleteChildProcess(t *testing.T) {
	// Clear the global state
	childMu.Lock()
	childProcesses = make(map[string]*exec.Cmd)
	childMu.Unlock()

	cmd := execCommand("echo", "test")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Register
	registerChildProcess("test-child", cmd)

	childMu.Lock()
	if _, ok := childProcesses["test-child"]; !ok {
		childMu.Unlock()
		t.Fatal("process not registered")
	}
	childMu.Unlock()

	// Delete
	deleteChildProcess(cmd.Process.Pid)

	childMu.Lock()
	if _, ok := childProcesses["test-child"]; ok {
		childMu.Unlock()
		t.Fatal("process should be deleted")
	}
	childMu.Unlock()

	// Wait for the process to finish
	cmd.Wait()
}

func TestDeleteChildProcess_NotFound(t *testing.T) {
	// Should not panic
	deleteChildProcess(999999)
}

func TestShutdownAllChildren(t *testing.T) {
	// Clear the global state
	childMu.Lock()
	childProcesses = make(map[string]*exec.Cmd)
	childMu.Unlock()

	// Start a long-running process
	cmd := execCommand("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	pid := cmd.Process.Pid
	registerChildProcess("long-sleep", cmd)

	// Shutdown should kill it
	shutdownAllChildren()

	// Give OS a moment to clean up
	time.Sleep(50 * time.Millisecond)

	// Process should be terminated — kill(pid, 0) returns error if gone
	err := syscall.Kill(pid, 0)
	if err == nil {
		t.Fatalf("process %d still alive after shutdown", pid)
	}
}
