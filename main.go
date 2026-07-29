package main

import (
	"os"

	"github.com/ALiwoto/codex-dedup/src/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
