package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	forceDocker := flag.Bool("docker", false, "Force Docker mode")
	forceBareMetal := flag.Bool("bare-metal", false, "Force bare-metal mode")
	flag.Parse()

	if *showVersion {
		fmt.Printf("coginit %s\n", version)
		os.Exit(0)
	}

	if *forceDocker && *forceBareMetal {
		log.Fatal("[coginit] Cannot specify both --docker and --bare-metal")
	}

	if *forceDocker {
		runDockerInit()
	} else if *forceBareMetal {
		runBareMetalInit()
	} else {
		if isDocker() {
			log.Println("[coginit] Detected Docker environment")
			runDockerInit()
		} else {
			log.Println("[coginit] Detected bare-metal environment")
			runBareMetalInit()
		}
	}
}

// isDocker checks if we're running inside a container.
func isDocker() bool {
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
