package coginit

import (
	"log"
	"os"
)

var version = "dev"

// Version returns the build version set via ldflags.
func Version() string { return version }

// Run is the main entry point for coginit.
// It auto-detects Docker vs bare-metal unless forceDocker/forceBareMetal is set.
func Run(forceDocker, forceBareMetal bool) {
	if forceDocker {
		runDockerInit()
	} else if forceBareMetal {
		runBareMetalInit()
	} else {
		if IsDocker() {
			log.Println("[coginit] Detected Docker environment")
			runDockerInit()
		} else {
			log.Println("[coginit] Detected bare-metal environment")
			runBareMetalInit()
		}
	}
}

// IsDocker checks if we're running inside a container.
func IsDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}
	if os.Getenv("container") != "" {
		return true
	}
	return false
}
