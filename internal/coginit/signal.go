//go:build linux

package coginit

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// startSignalReaper handles graceful shutdown.
// SIGCHLD is NOT handled here — process supervision goroutines
// (supervise) reap and restart each child individually.
// The onShutdown callback is called when SIGINT/SIGTERM is received
// after all children are terminated.
func startSignalReaper(onShutdown func()) {
	sigChan := make(chan os.Signal, 10)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for sig := range sigChan {
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			log.Println("[coginit] Shutdown signal received, terminating engines...")
			shutdownAllChildren()
			onShutdown()
		}
	}
}
