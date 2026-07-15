//go:build !linux

package main

// startSignalReaper is a no-op on non-Linux platforms.
func startSignalReaper(onShutdown func()) {}
