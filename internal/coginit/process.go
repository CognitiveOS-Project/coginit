//go:build linux

package coginit

import (
	"log"
	"os/exec"
	"sync"
	"syscall"
	"time"
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

// supervise watches a child process and restarts it on exit.
// The restartFn is called to create and start a new instance.
func supervise(name string, cmd *exec.Cmd, restartFn func() *exec.Cmd) {
	go func() {
		for {
			err := cmd.Wait()
			pid := 0
			if cmd.Process != nil {
				pid = cmd.Process.Pid
			}
			deleteChildProcess(pid)
			if err != nil {
				log.Printf("[coginit] %s exited: %v, restarting in 500ms", name, err)
			} else {
				log.Printf("[coginit] %s exited cleanly, restarting in 500ms", name)
			}
			time.Sleep(500 * time.Millisecond)

			cmd = restartFn()
			if cmd == nil {
				log.Printf("[coginit] Failed to restart %s, retrying in 1s", name)
				time.Sleep(1 * time.Second)
				continue
			}
		}
	}()
}

// shutdownAllChildren sends SIGTERM to all tracked processes.
func shutdownAllChildren() {
	childMu.Lock()
	defer childMu.Unlock()

	for name, cmd := range childProcesses {
		if cmd.Process != nil {
			log.Printf("[coginit] Terminating %s (PID %d)...", name, cmd.Process.Pid)
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
