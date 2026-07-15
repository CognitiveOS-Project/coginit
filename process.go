package main

import (
	"log"
	"os/exec"
	"sync"
	"syscall"
)

var (
	childProcesses = make(map[string]*exec.Cmd)
	childMu        sync.Mutex
)

// execCommand creates a command with SysProcAttr for process group management.
func execCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	return cmd
}

// registerChildProcess stores a process by name for later management.
func registerChildProcess(name string, cmd *exec.Cmd) {
	childMu.Lock()
	defer childMu.Unlock()
	childProcesses[name] = cmd
}

// deleteChildProcess removes a process from tracking.
func deleteChildProcess(pid int) {
	childMu.Lock()
	defer childMu.Unlock()
	for name, cmd := range childProcesses {
		if cmd.Process != nil && cmd.Process.Pid == pid {
			delete(childProcesses, name)
			return
		}
	}
}

// shutdownAllChildren sends SIGTERM to all tracked processes.
func shutdownAllChildren() {
	childMu.Lock()
	defer childMu.Unlock()

	for name, cmd := range childProcesses {
		if cmd.Process != nil {
			log.Printf("[coginit] Terminating %s (PID %d)...", name, cmd.Process.Pid)
			// Send SIGTERM to the process group
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		}
	}

	// Wait briefly for processes to exit
	for _, cmd := range childProcesses {
		if cmd.Process != nil {
			_ = cmd.Wait()
		}
	}

	log.Println("[coginit] All engines terminated")
}
