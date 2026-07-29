package coginit

import (
	"os"

	"github.com/BurntSushi/toml"
)

// DefaultModelPath resolves the raw model GGUF path using:
//  1. COGNITIVEOS_RAW_MODEL_PATH env var
//  2. /etc/cognitiveos/config.toml [raw_model].model
//  3. Hardcoded default
func DefaultModelPath() string {
	if p := os.Getenv("COGNITIVEOS_RAW_MODEL_PATH"); p != "" {
		return p
	}
	if p := readRawModelPathFromConfig("/etc/cognitiveos/config.toml"); p != "" {
		return p
	}
	return "/cognitiveos/models/raw/raw-model.gguf"
}

func readRawModelPathFromConfig(path string) string {
	var tc struct {
		RawModel struct {
			Model string `toml:"model"`
		} `toml:"raw_model"`
	}
	if _, err := toml.DecodeFile(path, &tc); err != nil {
		return ""
	}
	return tc.RawModel.Model
}
