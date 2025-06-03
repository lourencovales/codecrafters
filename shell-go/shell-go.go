package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

func main() {

	for {
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		commandNewLine, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input: %v", err)
		}
		command := strings.TrimSuffix(commandNewLine, "\n")

		switch {
		case strings.HasPrefix(command, "exit"):
			exit(command)
		case strings.HasPrefix(command, "echo"):
			echo(command)
		case strings.HasPrefix(command, "type"):
			typ(command)
		case strings.HasPrefix(command, "pwd"):
			pwd()
		case strings.HasPrefix(command, "cd"):
			cd(command)
		default:
			eval(command)
		}
	}
}

func exit(command string) {
	if command == "exit 0" {
		os.Exit(0)
	}
}

func echo(command string) {
	echo := strings.TrimPrefix(command, "echo ")
	fmt.Fprintf(os.Stdout, "%s\n", echo)
}

func typ(command string) {
	c := strings.TrimPrefix(command, "type ")
	if builtIns(c) {
		fmt.Printf("%s is a shell builtin\n", c)
	} else if b, s := inPath(c); b {
		fmt.Printf("%s is %s", c, s)
	} else {
		fmt.Println(c + ": not found")
	}
}

func pwd() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Errorf("Error gettings the working directory: %w", err)
	}
	fmt.Fprintf(os.Stdout, "%s\n", dir)
}

func cd(command string) {
	dir := strings.TrimPrefix(command, "cd ")
	if dir == "~" {
		dir = os.Getenv("HOME")
	}
	err := os.Chdir(dir)
	if err != nil {
		err = fmt.Errorf("cd: %s: No such file or directory", err.(*fs.PathError).Path)
		fmt.Fprintf(os.Stdout, "%s\n", err)
	}
}

func eval(command string) {
	commandArgs := strings.Split(command, " ")
	commandName := commandArgs[0]
	commandOpts := commandArgs[1:]
	if b, _ := inPath(commandName); b {
		cmd := exec.Command(commandName, commandOpts...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	} else {
		fmt.Println(commandName + ": command not found")
	}
}

func builtIns(command string) bool {

	knownComm := []string{
		"exit",
		"echo",
		"type",
		"pwd",
	}

	for _, c := range knownComm {
		if strings.Contains(command, c) {
			return true
		}
	}

	return false
}

func inPath(command string) (bool, string) {
	listPath := strings.Split(os.Getenv("PATH"), ":")
	for _, p := range listPath {
		files, err := os.ReadDir(p)
		if err != nil {
			fmt.Errorf("problem with dir")
		}
		for _, f := range files {
			if f.Name() == command {
				return true, fmt.Sprintf("%s/%s\n", p, f.Name())
			}
		}
	}
	return false, ""
}
