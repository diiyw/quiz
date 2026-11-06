// quiz-cli uses the same extraction core without a display server or CGO.
package main

import (
	"github.com/diiyw/quiz/internal/cli"
	"os"
)

func main() { os.Exit(cli.Run(os.Args[1:], nil, os.Stdout, os.Stderr)) }
