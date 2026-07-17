//go:build linux

package coginit

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// startSignalReaper handles zombie process reaping and graceful shutdown.
// The onShutdown callback is called when SIGINT/SIGTERM is received
// after all children are terminated.
func startSignalReaper(onShutdown func()) {
	sigChan := make(chan os.Signal, 10)
	signal.Notify(sigChan, syscall.SIGCHLD, syscall.SIGINT, syscall.SIGTERM)

	for sig := range sigChan {
		switch sig {
		case syscall.SIGCHLD:
			// Reap all terminated child processes
			for {
				var status syscall.WaitStatus
				pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
				if err != nil || pid <= 0 {
					break
				}
				log.Printf("[coginit] Reaped PID %d", pid)
				deleteChildProcess(pid)
			}

		case syscall.SIGINT, syscall.SIGTERM:
			log.Println("[coginit] Shutdown signal received, terminating engines...")
			shutdownAllChildren()
			onShutdown()
		}
	}
}
