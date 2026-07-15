//go:build linux

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	backdoorLogPath    = "/cognitiveos/logs/backdoor.log"
	scrollLockTimeout  = 2 * time.Second
	scrollLockRequired = 3
)

// comboState tracks currently pressed keys and scrolllock timing.
type comboState struct {
	mu             sync.Mutex
	pressed        map[uint16]bool
	scrollCount    int
	scrollFirst    time.Time
	onTrigger      func(combo string) // override in tests
}

func newComboState() *comboState {
	return &comboState{
		pressed: make(map[uint16]bool),
	}
}

// startBackdoorMonitor launches VT and serial backdoor goroutines.
func startBackdoorMonitor() {
	log.Println("[coginit] Starting backdoor monitor...")
	go startVTBackdoor()
	go startSerialBackdoor()
}

// startVTBackdoor watches evdev devices for keyboard combos.
func startVTBackdoor() {
	// Scan existing devices
	scanEvdevDevices()

	// Watch for hotplug via inotify
	watchInputDevices()
}

// scanEvdevDevices opens all keyboard devices in /dev/input/.
func scanEvdevDevices() {
	matches, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		log.Printf("[coginit] Warning: failed to scan /dev/input: %v", err)
		return
	}
	for _, path := range matches {
		if isKeyboard(path) {
			go readEvdevLoop(path)
		}
	}
}

// watchInputDevices uses inotify to detect new keyboards plugged in.
func watchInputDevices() {
	fd, err := syscall.InotifyInit1(0)
	if err != nil {
		log.Printf("[coginit] Warning: inotify init failed: %v", err)
		return
	}
	defer syscall.Close(fd)

	_, err = syscall.InotifyAddWatch(fd, "/dev/input", syscall.IN_CREATE)
	if err != nil {
		log.Printf("[coginit] Warning: inotify add watch failed: %v", err)
		return
	}

	var buf [4096]byte
	for {
		n, err := syscall.Read(fd, buf[:])
		if err != nil || n == 0 {
			log.Printf("[coginit] Warning: inotify read failed: %v", err)
			return
		}

		// Parse inotify events
		offset := 0
		for offset < n {
			wd := *(*int32)(unsafe.Pointer(&buf[offset]))
			mask := *(*uint32)(unsafe.Pointer(&buf[offset+4]))
			nameOffset := offset + 16

			if nameOffset > n {
				break
			}

			nameBytes := buf[nameOffset:n]
			// Find null terminator
			var name []byte
			for _, b := range nameBytes {
				if b == 0 {
					break
				}
				name = append(name, b)
			}

			_ = wd
			if mask&syscall.IN_CREATE != 0 && len(name) > 0 {
				path := filepath.Join("/dev/input", string(name))
				// Small delay for device to be ready
				time.Sleep(100 * time.Millisecond)
				if isKeyboard(path) {
					log.Printf("[coginit] Backdoor: new keyboard detected: %s", path)
					go readEvdevLoop(path)
				}
			}

			// Advance: 16-byte header + aligned name length
			nameLen := len(name) + 1
			offset += 16 + ((nameLen + 15) &^ 15)
		}
	}
}

// readEvdevLoop reads events from a single evdev device.
func readEvdevLoop(path string) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		log.Printf("[coginit] Warning: failed to open %s: %v", path, err)
		return
	}
	defer syscall.Close(fd)

	state := newComboState()
	var event inputEvent
	eventSize := int(unsafe.Sizeof(event))

	for {
		buf := make([]byte, eventSize)
		n, err := syscall.Read(fd, buf)
		if err != nil || n < eventSize {
			if err != syscall.EAGAIN {
				log.Printf("[coginit] Warning: evdev read error on %s: %v", path, err)
				return
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// Parse the event
		event.Type = *(*uint16)(unsafe.Pointer(&buf[16]))
		event.Code = *(*uint16)(unsafe.Pointer(&buf[18]))
		event.Value = *(*int32)(unsafe.Pointer(&buf[20]))

		if event.Type == evKey {
			state.processKeyEvent(event.Code, event.Value)
		}
	}
}

// processKeyEvent handles a single key event for combo detection.
func (s *comboState) processKeyEvent(code uint16, value int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch value {
	case keyPress:
		s.pressed[code] = true
		s.checkCombos(code)
	case keyRelease:
		delete(s.pressed, code)
	case keyHold:
		// Ignore autorepeat
	}
}

// checkCombos checks if any backdoor combo is matched.
func (s *comboState) checkCombos(code uint16) {
	// Combo 1: Ctrl+Alt+Backspace
	if s.pressed[keyLeftCtrl] && s.pressed[keyLeftAlt] && s.pressed[keyBackspace] {
		s.triggerBackdoor("Ctrl+Alt+Backspace")
		return
	}

	// Combo 2: Alt+PrintScreen+K
	if s.pressed[keyLeftAlt] && s.pressed[keySysRq] && s.pressed[keyK] {
		s.triggerBackdoor("Alt+PrintScreen+K")
		return
	}

	// Combo 3: Ctrl+Alt+Del
	if s.pressed[keyLeftCtrl] && s.pressed[keyLeftAlt] && s.pressed[keyDelete] {
		s.triggerBackdoor("Ctrl+Alt+Del")
		return
	}

	// Combo 4: ScrollLock ×3 within 2 seconds
	if code == keyScrollLock {
		now := time.Now()
		if s.scrollCount == 0 || now.Sub(s.scrollFirst) > scrollLockTimeout {
			s.scrollFirst = now
			s.scrollCount = 1
		} else {
			s.scrollCount++
		}
		if s.scrollCount >= scrollLockRequired {
			s.scrollCount = 0
			s.triggerBackdoor("ScrollLockx3")
			return
		}
	}

	// Combo 5: Super+Grave
	if s.pressed[keyLeftMeta] && s.pressed[keyGrave] {
		s.triggerBackdoor("Super+Grave")
	}
}

// triggerBackdoor spawns the VT2 shell and logs the access.
func (s *comboState) triggerBackdoor(combo string) {
	// Allow test override
	if s.onTrigger != nil {
		s.onTrigger(combo)
		return
	}

	logAccess(combo, "vt2")

	log.Printf("[coginit] BACKDOOR: combo %s triggered, switching to VT2", combo)

	if err := vtActivate(2); err != nil {
		log.Printf("[coginit] BACKDOOR: failed to activate VT2: %v", err)
		return
	}
	if err := vtWaitActive(2); err != nil {
		log.Printf("[coginit] BACKDOOR: failed to wait for VT2: %v", err)
		return
	}

	spawnShell("/dev/tty2", combo)

	// Switch back to TUI
	if err := vtActivate(1); err != nil {
		log.Printf("[coginit] BACKDOOR: failed to switch back to VT1: %v", err)
	}
}

// spawnShell runs /bin/sh on the given tty device.
func spawnShell(tty string, combo string) {
	f, err := os.OpenFile(tty, os.O_RDWR, 0)
	if err != nil {
		log.Printf("[coginit] BACKDOOR: failed to open %s: %v", tty, err)
		return
	}
	defer f.Close()

	cmd := exec.Command("/bin/sh")
	cmd.Stdin = f
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    int(f.Fd()),
	}

	fmt.Fprintf(f, "\r\nBACKDOOR SHELL — CognitiveOS firmware override\r\n")
	fmt.Fprintf(f, "Combo: %s | TTY: %s\r\n", combo, tty)
	fmt.Fprintf(f, "Type 'exit' to return to TUI.\r\n\r\n")

	if err := cmd.Run(); err != nil {
		log.Printf("[coginit] BACKDOOR: shell on %s exited: %v", tty, err)
	}
}

// logAccess writes a backdoor access entry to the log file.
func logAccess(combo string, channel string) {
	f, err := os.OpenFile(backdoorLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		log.Printf("[coginit] BACKDOOR: failed to open log: %v", err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "%s combo=%s channel=%s\n", time.Now().UTC().Format(time.RFC3339), combo, channel)
}

// startSerialBackdoor watches /dev/ttyAMA0 for input and spawns shells.
func startSerialBackdoor() {
	serialPath := "/dev/ttyAMA0"

	// Check if serial device exists
	if _, err := os.Stat(serialPath); os.IsNotExist(err) {
		log.Println("[coginit] Backdoor: serial disabled (no /dev/ttyAMA0)")
		return
	}

	log.Println("[coginit] Backdoor: serial console available on /dev/ttyAMA0")

	for {
		f, err := os.OpenFile(serialPath, os.O_RDWR, 0)
		if err != nil {
			log.Printf("[coginit] BACKDOOR: failed to open %s: %v", serialPath, err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Read one byte to detect activity
		buf := make([]byte, 1)
		_, err = f.Read(buf)
		if err != nil {
			f.Close()
			time.Sleep(time.Second)
			continue
		}

		logAccess("serial-connect", "uart")

		log.Println("[coginit] BACKDOOR: serial activity detected, spawning shell")

		cmd := exec.Command("/bin/sh")
		cmd.Stdin = f
		cmd.Stdout = f
		cmd.Stderr = f
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid:  true,
			Setctty: true,
			Ctty:    int(f.Fd()),
		}

		fmt.Fprintf(f, "\r\nBACKDOOR SHELL — CognitiveOS firmware override\r\n")
		fmt.Fprintf(f, "Channel: UART /dev/ttyAMA0\r\n")
		fmt.Fprintf(f, "Type 'exit' to disconnect.\r\n\r\n")

		// Keep serial fd open while shell runs
		cmd.Run()
		f.Close()

		log.Println("[coginit] BACKDOOR: serial shell exited, watching for next connection")
	}
}
