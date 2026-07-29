package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	coginit "github.com/CognitiveOS-Project/coginit/internal/coginit"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	forceDocker := flag.Bool("docker", false, "Force Docker mode")
	forceBareMetal := flag.Bool("bare-metal", false, "Force bare-metal mode")
	modelPath := flag.String("model", "/cognitiveos/models/raw/raw-model.gguf", "Path to raw model GGUF file")
	flag.Parse()

	if *showVersion {
		fmt.Printf("coginit %s\n", coginit.Version())
		os.Exit(0)
	}

	if *forceDocker && *forceBareMetal {
		log.Fatal("[coginit] Cannot specify both --docker and --bare-metal")
	}

	coginit.SetModelPath(*modelPath)
	coginit.Run(*forceDocker, *forceBareMetal)
}
