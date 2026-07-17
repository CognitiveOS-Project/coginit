//go:build linux

package coginit

import (
	"os"
	"syscall"
	"unsafe"
)

// VT ioctl numbers (from linux/tty.h)
const (
	ioctlVTActivate   = 0x5602
	ioctlVTWaitActive = 0x5603
)

// evdev ioctl numbers (from linux/input.h)
const (
	evKey        = 0x01
	evSyn        = 0x00
	keyPress     = 1
	keyRelease   = 0
	keyHold      = 2
	evKeyBitSize = 32 // bytes for EV_KEY bitmask (keys 0-255)
)

// Key codes (from linux/input-event-codes.h)
const (
	keyLeftCtrl    = 29
	keyLeftAlt     = 56
	keyBackspace   = 14
	keySysRq       = 99
	keyK           = 37
	keyDelete      = 111
	keyScrollLock  = 70
	keyLeftMeta    = 125
	keyGrave       = 41
)

// inputEvent mirrors struct input_event from linux/input.h.
// Size: 24 bytes on 64-bit (16-byte timeval + 2+2+4).
type inputEvent struct {
	Timeval syscall.Timeval
	Type    uint16
	Code    uint16
	Value   int32
}

// eviocgbit returns the ioctl code for EVIOCGBIT(type, nbytes).
func eviocgbit(evtype, nbytes int) uintptr {
	return uintptr(0x4200 + evtype + (nbytes << 8))
}

// vtActivate switches the active virtual terminal.
func vtActivate(num int) error {
	fd, err := os.OpenFile("/dev/tty0", os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), ioctlVTActivate, uintptr(num))
	if errno != 0 {
		return errno
	}
	return nil
}

// vtWaitActive blocks until virtual terminal num is active.
func vtWaitActive(num int) error {
	fd, err := os.OpenFile("/dev/tty0", os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), ioctlVTWaitActive, uintptr(num))
	if errno != 0 {
		return errno
	}
	return nil
}

// isKeyboard checks if an evdev device supports EV_KEY events.
func isKeyboard(path string) bool {
	fd, err := syscall.Open(path, syscall.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer syscall.Close(fd)

	// Query EV_KEY bitmask: EVIOCGBIT(EV_KEY, 32 bytes)
	var bitmap [evKeyBitSize]byte
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		eviocgbit(evKey, evKeyBitSize),
		uintptr(unsafe.Pointer(&bitmap[0])),
	)
	if errno != 0 {
		return false
	}

	// Check if any bit is set — device supports at least one key
	for _, b := range bitmap {
		if b != 0 {
			return true
		}
	}
	return false
}
