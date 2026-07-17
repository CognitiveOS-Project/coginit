//go:build !linux

package coginit

// startSignalReaper is a no-op on non-Linux platforms.
func startSignalReaper(onShutdown func()) {}
