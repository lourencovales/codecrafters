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
	quoteChar rune
	answer    string
	full      string
}

func main() {

	for {
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		commandNewLine, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input: %v", err)
		}
		commOrig := strings.TrimSuffix(commandNewLine, "\n")
		comm := strings.SplitAfterN(commOrig, " ", 2)

		command := &Command{}

		command.cmd = comm[0][:len(comm[0])-1]
		if len(comm) > 1 {
			command.args = append(command.args, comm[1])
		}
		command.full = commOrig

		//if command.builtIns() {
		//	fmt.Fprintln(os.Stdout, "%s\n", command.answer)
		//}

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
	if command.full == "exit 0" {
		os.Exit(0)
	}
}

func (command *Command) echo() {
	command.parseQuoteChar()
	if command.quoteChar != rune(0) {
		command.echoParseArgs()
		command.answer = strings.Join(command.args, " ")
	} else {
		command.answer = strings.Join(command.args, " ")
	}
}

func (command *Command) typ() {
	if command.builtIn() {
		command.answer = fmt.Sprintf("%s is a shell builtin", command.args[0])
	} else if b, s := command.inPath(); b {
		command.answer = fmt.Sprintf("%s is %s", command.args[0], s)
	}
	command.answer = command.args[0] + ": not found"

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
		fmt.Errorf("cd: %s: No such file or directory", err.(*fs.PathError).Path)
	}
	command.answer = ""
}

func (command *Command) eval() {
	command.parseQuoteChar()
	if command.quoteChar != rune(0) {
		command.evalParseArgs()
	}
	if b, _ := command.inPath(); b {
		cm := exec.Command(command.cmd, command.args...)
		cm.Stdout = os.Stdout
		cm.Stderr = os.Stderr
		cm.Run()
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
		if strings.Contains(c, command.args[0]) {
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
			if f.Name() == command.cmd {
				return true, fmt.Sprintf("%s/%s\n", p, f.Name())
			}
		}
	}
	return false, ""
}

// Thanks to https://github.com/mjl-/tokenize for the inspiration
func (command *Command) evalParseArgs() {
	r := make([]rune, 0)
	prevQuote := false
	escape := false
	result := []string{}
	for _, c := range command.args[0] {

		switch c {
		case command.quoteChar:
			if escape {
				r = append(r, c)
				escape = false
			} else if !prevQuote {
				prevQuote = true
			} else if prevQuote {
				result = append(result, string(r))
				r = []rune{}
				prevQuote = false
			}
		case ' ':
			if prevQuote {
				r = append(r, c)
			}
		case '\\':
			if escape {
				r = append(r, c)
				escape = false
			} else {
				escape = true
			}
		default:
			r = append(r, c)
			escape = false
		}
	}

	command.args = result
}

func (command *Command) echoParseArgs() {
	r := make([]rune, 0)
	prevQuote := false
	escape := false
	result := []string{}

	for _, c := range command.args[0] {
		switch c {
		case command.quoteChar:
			if escape {
				r = append(r, c)
			}
			if prevQuote {
				result = append(result, string(r))
				r = []rune{}
				prevQuote = false
			} else {
				prevQuote = true
			}
		case ' ':
			if prevQuote {
				r = append(r, c)
			}
		case '\\':
			if prevQuote {
				r = append(r, c)
				escape = false
			} else {
				escape = true
			}
		default:
			r = append(r, c)
			escape = false
		}
		if len(r) != 0 {
			result = append(result, string(r))
		}
	}

	command.args = result
}

func (command *Command) parseQuoteChar() {
	command.quoteChar = rune(0)
	for _, c := range command.args[0] {
		if c == quote {
			command.quoteChar = quote
			return
		} else if c == doubleQuote {
			command.quoteChar = doubleQuote
			return
		}
	}
}
