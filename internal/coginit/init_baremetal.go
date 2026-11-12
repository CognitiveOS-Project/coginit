//go:build linux

package coginit

import (
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func runBareMetalInit() {
	log.Println("[coginit] Starting in bare-metal mode")

	// 1. Mount virtual filesystems
	mountVirtualFS()

	// 2. Start signal reaper (handles zombies + graceful shutdown)
	go startSignalReaper(powerOff)

	// 3. Configure loopback network
	configureLoopback()

	// 4. Create runtime directories
	createRuntimeDirs()

	// 5. Install boot-stage dependencies
	installDependencies("boot")

	// 6. Start engines in order with supervision
	cmd := startCograw()
	waitForSocket("/cognitiveos/run/raw.sock", defaultSocketTimeout())
	if cmd != nil {
		supervise("cograw", cmd, startCograw)
	}

	cmd = startCoginfer()
	waitForHTTP("http://127.0.0.1:11434/health", defaultHTTPTimeout)
	if cmd != nil {
		supervise("coginfer", cmd, startCoginfer)
	}

	cmd = startCognitiveosd()
	waitForSocket("/cognitiveos/run/daemon.sock", defaultSocketTimeout())
	if cmd != nil {
		supervise("cognitiveosd", cmd, startCognitiveosd)
	}

	// 7. Install runtime-stage dependencies
	installDependencies("runtime")

	log.Println("[coginit] All engines started. Starting backdoor monitor...")

	// 8. Start backdoor monitor (keyboard combos + serial)
	go startBackdoorMonitor()

	// 9. TUI supervision loop (restarts CLI on crash)
	tuiSupervisionLoop()
}

// mountVirtualFS mounts essential virtual filesystems for bare-metal operation.
func mountVirtualFS() {
	mounts := []struct {
		source string
		target string
		fstype string
		flags  uintptr
	}{
		{"proc", "/proc", "proc", 0},
		{"sysfs", "/sys", "sysfs", 0},
		{"tmpfs", "/tmp", "tmpfs", 0},
		{"tmpfs", "/run", "tmpfs", 0},
		{"devpts", "/dev/pts", "devpts", 0},
	}

	for _, m := range mounts {
		if err := os.MkdirAll(m.target, 0755); err != nil {
			log.Printf("[coginit] Warning: mkdir %s failed: %v", m.target, err)
			continue
		}
		if err := syscall.Mount(m.source, m.target, m.fstype, m.flags, ""); err != nil {
			// EBUSY means already mounted — not a fatal error
			if err != syscall.EBUSY {
				log.Printf("[coginit] Warning: mount %s failed: %v", m.target, err)
			}
		} else {
			log.Printf("[coginit] Mounted %s", m.target)
		}
	}
}

// configureLoopback brings up the loopback network interface.
func configureLoopback() {
	log.Println("[coginit] Configuring loopback network...")
	cmd := execCommand("ip", "link", "set", "lo", "up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[coginit] Warning: failed to bring up loopback: %v", err)
		return
	}
	cmd = execCommand("ip", "addr", "add", "127.0.0.1/8", "dev", "lo")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// EEXIST means already configured — fine
		log.Printf("[coginit] Warning: failed to add loopback addr: %v", err)
	}
	log.Println("[coginit] Loopback network configured")
}

// tuiSupervisionLoop restarts cognitiveos-cli if it exits.
func tuiSupervisionLoop() {
	for {
		log.Println("[coginit] Starting cognitiveos-cli...")

		cmd := execCommand("/usr/local/bin/cognitiveos-cli")
		cmd.Stdin = openTTY()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			log.Printf("[coginit] Failed to start CLI: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		registerChildProcess("cognitiveos-cli", cmd)
		err := cmd.Wait()
		deleteChildProcess(cmd.Process.Pid)

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					log.Printf("[coginit] CLI exited with status %d", status.ExitStatus())
				}
			} else {
				log.Printf("[coginit] CLI exited: %v", err)
			}
		} else {
			log.Println("[coginit] CLI exited cleanly")
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// openTTY opens /dev/tty1 for the CLI.
func openTTY() *os.File {
	f, err := os.OpenFile("/dev/tty1", os.O_RDWR, 0)
	if err != nil {
		log.Printf("[coginit] Warning: failed to open /dev/tty1: %v", err)
		return os.Stdin
	}
	return f
}

// powerOff initiates a clean system shutdown.
func powerOff() {
	log.Println("[coginit] Powering off...")
	// Sync filesystems
	syscall.Sync()
	// Power off
	_ = syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}
