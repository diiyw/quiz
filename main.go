//go:build !headless

package main

import (
	"os"
	"runtime"

	"github.com/diiyw/quiz/internal/cli"
	"github.com/diiyw/quiz/internal/desktop"
)

func main() {
	runtime.LockOSThread()
	os.Exit(cli.Run(os.Args[1:], desktop.Run, os.Stdout, os.Stderr))
}
