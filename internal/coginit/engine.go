package coginit

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	defaultHTTPTimeout   = 15 * time.Second
	maxLogSizeBytes      = 10 * 1024 * 1024 // 10MB — rotate when exceeded
)

// defaultSocketTimeout returns the socket wait timeout, configurable via
// COGINIT_SOCKET_TIMEOUT env var (e.g. "30s", "1m"). Falls back to 10s.
func defaultSocketTimeout() time.Duration {
	d := 10 * time.Second
	if v := os.Getenv("COGINIT_SOCKET_TIMEOUT"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			d = parsed
		}
	}
	return d
}

// modelPath is the path to the raw model GGUF file.
// Override via coginit --model flag or SetModelPath().
var modelPath = "/cognitiveos/models/raw/raw-model.gguf"

// SetModelPath overrides the default raw model path.
func SetModelPath(path string) {
	modelPath = path
}

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
// Returns the command, or nil if start failed.
func startCograw() *exec.Cmd {
	log.Println("[coginit] Starting cograw...")

	args := []string{
		"--model", modelPath,
		"--socket", "/cognitiveos/run/raw.sock",
	}

	logFile := openLogFile("cograw")
	cmd := execCommand("/usr/local/bin/cograw", args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		log.Printf("[coginit] Failed to start cograw: %v", err)
		return nil
	}

	registerChildProcess("cograw", cmd)
	log.Printf("[coginit] cograw started (PID %d)", cmd.Process.Pid)
	return cmd
}

// startCoginfer starts the wide model inference server.
// Returns the command, or nil if start failed.
func startCoginfer() *exec.Cmd {
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
		return nil
	}

	registerChildProcess("coginfer", cmd)
	log.Printf("[coginit] coginfer started (PID %d)", cmd.Process.Pid)
	return cmd
}

// startCognitiveosd starts the system daemon.
// Returns the command, or nil if start failed.
func startCognitiveosd() *exec.Cmd {
	log.Println("[coginit] Starting cognitiveosd...")

	logFile := openLogFile("cognitiveosd")
	cmd := execCommand("/usr/local/bin/cognitiveosd")
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		log.Printf("[coginit] Failed to start cognitiveosd: %v", err)
		return nil
	}

	registerChildProcess("cognitiveosd", cmd)
	log.Printf("[coginit] cognitiveosd started (PID %d)", cmd.Process.Pid)
	return cmd
}

// openLogFile opens or creates a log file for a service.
// If the existing log exceeds maxLogSizeBytes, it is rotated to name.log.1.
func openLogFile(name string) *os.File {
	path := filepath.Join("/cognitiveos/logs", name+".log")

	// Rotate if log is too large
	if info, err := os.Stat(path); err == nil && info.Size() > maxLogSizeBytes {
		backup := path + ".1"
		_ = os.Remove(backup) // remove previous backup
		if err := os.Rename(path, backup); err != nil {
			log.Printf("[coginit] Warning: failed to rotate %s: %v", path, err)
		} else {
			log.Printf("[coginit] Rotated %s (%d bytes)", path, info.Size())
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[coginit] Warning: failed to open log file %s: %v", path, err)
		return os.Stderr
	}
	return f
}
