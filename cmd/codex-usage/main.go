package main

import (
	"os"

	"github.com/ilcm96/codex-usage/internal/cli/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
