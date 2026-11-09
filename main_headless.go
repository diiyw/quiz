//go:build headless

package main

import (
	"os"

	"github.com/diiyw/quiz/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], nil, os.Stdout, os.Stderr))
}
