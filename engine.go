package main

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultSocketTimeout = 10 * time.Second
	defaultHTTPTimeout   = 15 * time.Second
)

// createRuntimeDirs ensures required directories exist.
func createRuntimeDirs() {
	dirs := []string{"/cognitiveos/run", "/cognitiveos/logs"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("[coginit] Warning: failed to create %s: %v", dir, err)
		}
	}
	log.Println("[coginit] Runtime directories ready")
}

// installDependencies runs cpm install-dependencies for the given stage.
func installDependencies(stage string) {
	cpmPath := "/usr/local/bin/cpm"
	if _, err := os.Stat(cpmPath); os.IsNotExist(err) {
		log.Printf("[coginit] Warning: cpm not found at %s, skipping %s dependencies", cpmPath, stage)
		return
	}

	log.Printf("[coginit] Installing %s-stage dependencies...", stage)
	cmd := execCommand(cpmPath, "install-dependencies", "--stage", stage)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[coginit] Warning: %s-stage dependency installation failed: %v", stage, err)
	} else {
		log.Printf("[coginit] %s-stage dependencies installed", stage)
	}
}

// startCograw starts the raw model guardrail server.
func startCograw() {
	log.Println("[coginit] Starting cograw...")

	args := []string{
		"--model", "/cognitiveos/models/raw/raw-model.gguf",
		"--socket", "/cognitiveos/run/raw.sock",
	}

	logFile := openLogFile("cograw")
	cmd := execCommand("/usr/local/bin/cograw", args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		log.Printf("[coginit] Failed to start cograw: %v", err)
		return
	}

	registerChildProcess("cograw", cmd)
	log.Printf("[coginit] cograw started (PID %d)", cmd.Process.Pid)
}

// startCoginfer starts the wide model inference server.
func startCoginfer() {
	log.Println("[coginit] Starting coginfer...")

	args := []string{
		"--backend", "cgo",
		"--models", "/cognitiveos/models",
	}

	logFile := openLogFile("coginfer")
	cmd := execCommand("/usr/local/bin/coginfer", args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		log.Printf("[coginit] Failed to start coginfer: %v", err)
		return
	}

	registerChildProcess("coginfer", cmd)
	log.Printf("[coginit] coginfer started (PID %d)", cmd.Process.Pid)
}

// startCognitiveosd starts the system daemon.
func startCognitiveosd() {
	log.Println("[coginit] Starting cognitiveosd...")

	logFile := openLogFile("cognitiveosd")
	cmd := execCommand("/usr/local/bin/cognitiveosd")
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		log.Printf("[coginit] Failed to start cognitiveosd: %v", err)
		return
	}

	registerChildProcess("cognitiveosd", cmd)
	log.Printf("[coginit] cognitiveosd started (PID %d)", cmd.Process.Pid)
}

// openLogFile opens or creates a log file for a service.
func openLogFile(name string) *os.File {
	path := filepath.Join("/cognitiveos/logs", name+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("[coginit] Warning: failed to open log file %s: %v", path, err)
		return os.Stderr
	}
	return f
}
