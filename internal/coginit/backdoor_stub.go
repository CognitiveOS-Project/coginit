//go:build !linux

package coginit

// startBackdoorMonitor is a no-op on non-Linux platforms.
func startBackdoorMonitor() {}
