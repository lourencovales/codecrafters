package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

const (
	quote       = '\''
	doubleQuote = '"'
	escape      = '\\'
)

type Command struct {
	cmd       string
	args      []string
	isEscaped bool
	isQuoted  bool
	quoteChar rune
	answer    string
}

func main() {

	for {
		command := &Command{}

		fmt.Fprint(os.Stdout, "$ ")

		commandNewLine, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input: %v", err)
		}
		command.cmd = strings.TrimSuffix(commandNewLine, "\n")
		command.parseArgs()

		switch command.cmd {
		case "exit":
			command.exit()
		case "echo":
			command.echo()
		case "type":
			command.typ()
		case "pwd":
			command.pwd()
		case "cd":
			command.cd()
		default:
			command.eval()
		}
		if command.answer != "" {
			fmt.Printf("%s\n", command.answer)
		}
	}
}

func (command *Command) exit() {
	if command.args[0] == "0" {
		os.Exit(0)
	}
}

func (command *Command) echo() {
	command.answer = strings.Join(command.args, " ")
}

func (command *Command) typ() {
	command.answer = command.args[0] + ": not found"

	if command.builtIn() {
		command.answer = fmt.Sprintf("%s is a shell builtin", command.args[0])
	} else if b, s := command.inPath(); b {
		command.answer = fmt.Sprintf("%s is %s", command.args[0], s)
	}
}

func (command *Command) pwd() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Errorf("Error gettings the working directory: %w", err)
	}
	command.answer = dir
}

func (command *Command) cd() {
	if command.args[0] == "~" {
		command.args[0] = os.Getenv("HOME")
	}
	err := os.Chdir(command.args[0])
	if err != nil {
		e := fmt.Errorf("cd: %s: No such file or directory", err.(*fs.PathError).Path)
		command.answer = fmt.Sprint(e.Error())
		return
	}
	command.answer = ""
}

func (command *Command) eval() {
	if command.args != nil {
		if b, _ := command.inPath(); b {
			cm := exec.Command(command.cmd, command.args...)
			cm.Stdout = os.Stdout
			cm.Stderr = os.Stderr
			cm.Run()
		}
	} else {
		command.answer = command.cmd + ": command not found"
	}
}

func (command *Command) builtIn() bool {

	knownComm := []string{
		"exit",
		"echo",
		"type",
		"pwd",
	}

	for _, c := range knownComm {
		if c == command.args[0] {
			return true
		}
	}

	return false
}

func (command *Command) inPath() (bool, string) {
	listPath := strings.Split(os.Getenv("PATH"), ":")
	for _, p := range listPath {
		files, err := os.ReadDir(p)
		if err != nil {
			fmt.Errorf("problem with dir")
		}
		for _, f := range files {
			if f.Name() == command.cmd || f.Name() == command.args[0] {
				return true, fmt.Sprintf("%s/%s", p, f.Name())
			}
		}
	}
	return false, ""
}

// Thanks to https://github.com/mjl-/tokenize for the inspiration
func (command *Command) parseArgs() {
	r := make([]rune, 0)
	result := []string{}
	inSingleQuote := false
	inDoubleQuote := false

	peakForward := func(i int) rune {
		if len(command.cmd) > i+1 {
			return rune(command.cmd[i+1])
		}
		return rune(0)
	}

	for i, c := range command.cmd {
		switch c {
		case doubleQuote:
			if inSingleQuote || command.isEscaped {
				r = append(r, c)
				command.isEscaped = false
				continue
			}

			inDoubleQuote = !inDoubleQuote
		case quote:
			if inDoubleQuote || command.isEscaped {
				command.isEscaped = false
				r = append(r, c)
				continue
			}

			inSingleQuote = !inSingleQuote
		case ' ':
			if command.isEscaped || inDoubleQuote || inSingleQuote {
				command.isEscaped = false
				r = append(r, c)
				continue
			}
			if len(r) > 0 {
				result = append(result, string(r))
				r = []rune{}
			}
		case escape:
			if inSingleQuote || command.isEscaped {
				r = append(r, c)
				command.isEscaped = false
				continue
			}
			if inDoubleQuote {
				specialChars := []rune{'$', doubleQuote, escape}
				for _, char := range specialChars {
					if peakForward(i) == char {
						command.isEscaped = true
						break
					}
				}
				if !strings.HasPrefix(command.cmd, "echo") {
					r = append(r, c)
				}
			}
			if !inDoubleQuote && !inSingleQuote {
				command.isEscaped = true
			}

		default:
			r = append(r, c)
			command.isEscaped = false
		}
	}

	if len(r) > 0 {
		result = append(result, string(r))
	}

	command.cmd = result[0]
	if len(result) > 1 {
		command.args = result[1:]
	}
}
