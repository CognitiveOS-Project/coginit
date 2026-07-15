//go:build !linux

package main

// startBackdoorMonitor is a no-op on non-Linux platforms.
func startBackdoorMonitor() {}
