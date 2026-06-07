package codex

import (
	"os"
	"path/filepath"
)

const codexHomeEnv = "CODEX_HOME"

func ResolveCodexHome() string {
	if v := os.Getenv(codexHomeEnv); v != "" {
		return filepath.Clean(v)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".codex")
}
