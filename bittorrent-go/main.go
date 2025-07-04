package main

import (
	"fmt"
	"os"

	"github.com/lourencovales/codecrafters/bittorrent-go/cmd"
)

func main() {

	if len(os.Args) < 2 { // better to fail fast
		fmt.Fprintf(os.Stderr, "Error: a command is needed\n") // TODO
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	if err := cmd.Run(command, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
