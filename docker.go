package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func runDockerInit() {
	log.Println("[coginit] Starting in Docker mode")

	// 1. Start signal reaper
	go startSignalReaper()

	// 2. Create runtime directories
	createRuntimeDirs()

	// 3. Install boot-stage dependencies
	installDependencies("boot")

	// 4. Start engines in order
	startCograw()
	waitForSocket("/cognitiveos/run/raw.sock", defaultSocketTimeout)

	startCoginfer()
	waitForHTTP("http://127.0.0.1:11434/health", defaultHTTPTimeout)

	startCognitiveosd()
	waitForSocket("/cognitiveos/run/daemon.sock", defaultSocketTimeout)

	// 5. Install runtime-stage dependencies
	installDependencies("runtime")

	log.Println("[coginit] All engines started. Handing off to CLI...")

	// 6. Exec CLI (replaces coginit process)
	cliPath := "/usr/local/bin/cognitiveos-cli"
	args := []string{"cognitiveos-cli"}
	if len(os.Args) > 1 {
		args = append(args, os.Args[1:]...)
	}

	if err := syscall.Exec(cliPath, args, os.Environ()); err != nil {
		log.Fatalf("[coginit] Failed to exec CLI: %v", err)
	}
}

// startSignalReaper handles zombie process reaping and graceful shutdown.
func startSignalReaper() {
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
			os.Exit(0)
		}
	}
}
