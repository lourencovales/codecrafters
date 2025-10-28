package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	quote       = '\''
	doubleQuote = '"'
	escape      = '\\'
)

type Command struct {
	cmd         string
	args        []string
	isEscaped   bool
	isQuoted    bool
	quoteChar   rune
	answer      string
	stdoutRedir bool
	redirPath   string
	stderrRedir bool
	appendRedir bool
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
		if !command.stdoutRedir {
			if command.answer != "" {
				if !strings.HasSuffix(command.answer, "\n") {
					command.answer = command.answer + "\n"
				}
				fmt.Printf("%s", command.answer)
			}
		}

	}
}

func (command *Command) exit() {
	if command.args[0] == "0" {
		os.Exit(0)
	}
}

func (command *Command) echo() {
	output := strings.Join(command.args, " ")

	dir := filepath.Dir(command.redirPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		panic(err)
	}

	if command.stdoutRedir {
		if command.appendRedir {
			fileInfo, err := os.Stat(command.redirPath)
			needsNewline := err == nil && fileInfo.Size() > 0

			f, errOpen := os.OpenFile(command.redirPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if errOpen != nil {
				panic(errOpen)
			}

			if needsNewline {
				f.Write([]byte("\n"))
			}

			if _, errWrite := f.Write([]byte(output)); errWrite != nil {
				panic(errWrite)
			}
			if errClose := f.Close(); errClose != nil {
				panic(errClose)
			}
		} else if err := os.WriteFile(command.redirPath, []byte(output), 0644); err != nil {
			panic(err)
		}
	}

	if command.stderrRedir {
		if command.appendRedir {
			f, errOpen := os.OpenFile(command.redirPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if errOpen != nil {
				panic(errOpen)
			}
			if _, errWrite := f.Write([]byte("")); errWrite != nil { // TODO: ugly hack
				panic(errWrite)
			}
			if errClose := f.Close(); errClose != nil {
				panic(errClose)
			}
		} else if errWrite := os.WriteFile(command.redirPath, []byte(""), 0644); errWrite != nil { // TODO: ugly hack
			panic(errWrite)
		}
	}

	command.answer = output
}

func (command *Command) typ() {
	command.answer = command.args[0] + ": not found"

	if command.builtIn() {
		command.answer = fmt.Sprintf("%s is a shell builtin", command.args[0])
	} else if b, s := command.inPath(command.args[0]); b {
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
	if b, _ := command.inPath(command.cmd); b {
		cm := exec.Command(command.cmd, command.args...)

		var stdout, stderr bytes.Buffer
		cm.Stdout = &stdout
		cm.Stderr = &stderr

		err := cm.Run()

		dir := filepath.Dir(command.redirPath)
		if errDir := os.MkdirAll(dir, 0750); errDir != nil {
			panic(errDir)
		}

		if command.stdoutRedir {
			if command.appendRedir {
				fileInfo, err := os.Stat(command.redirPath)
				needsNewline := err == nil && fileInfo.Size() > 0

				f, errOpen := os.OpenFile(command.redirPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if errOpen != nil {
					panic(errOpen)
				}

				stdoutBytes := bytes.TrimSuffix(stdout.Bytes(), []byte("\n"))

				if needsNewline {
					f.Write([]byte("\n"))
				}

				if _, errWrite := f.Write(stdoutBytes); errWrite != nil {
					panic(errWrite)
				}
				if errClose := f.Close(); errClose != nil {
					panic(errClose)
				}
			} else if errWrite := os.WriteFile(command.redirPath, stdout.Bytes(), 0644); errWrite != nil {
				panic(errWrite)
			}
		}
		if command.stderrRedir {
			if command.appendRedir {
				fileInfo, err := os.Stat(command.redirPath)
				needsNewline := err == nil && fileInfo.Size() > 0

				f, errOpen := os.OpenFile(command.redirPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if errOpen != nil {
					panic(errOpen)
				}

				stderrBytes := bytes.TrimSuffix(stderr.Bytes(), []byte("\n"))

				if needsNewline {
					f.Write([]byte("\n"))
				}

				if _, errWrite := f.Write(stderrBytes); errWrite != nil {
					panic(errWrite)
				}
				if errClose := f.Close(); errClose != nil {
					panic(errClose)
				}
			} else if errWrite := os.WriteFile(command.redirPath, stderr.Bytes(), 0644); errWrite != nil {
				panic(errWrite)
			}

		}
		if err != nil && !command.stderrRedir {
			command.answer = stderr.String()
			command.stdoutRedir = false
			return
			panic(err)
		}
		command.answer = stdout.String()
		return

	}

	command.answer = command.cmd + ": command not found"
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

func (command *Command) inPath(name string) (bool, string) {
	listPath := strings.Split(os.Getenv("PATH"), ":")
	for _, p := range listPath {
		fullPath := filepath.Join(p, name)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
			return true, fullPath
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

	peakBackward := func(i int) rune {
		if i > 0 {
			return rune(command.cmd[i-1])
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
		case '>':
			if peakBackward(i) == '1' || peakBackward(i) == '2' {
				continue
			} else if peakBackward(i) == ' ' && peakForward(i) == ' ' {
				command.stdoutRedir = true
			} else if peakBackward(i) == '>' {
				command.appendRedir = true
				continue
			} else if peakForward(i) == '>' {
				command.appendRedir = true
				command.stdoutRedir = true
			} else {
				r = append(r, c)
			}
		case '1':
			if !(peakForward(i) == '>') {
				r = append(r, c)
			} else {
				command.stdoutRedir = true
			}
		case '2':
			if !(peakForward(i) == '>') {
				r = append(r, c)
			} else {
				command.stderrRedir = true
			}
		default:
			r = append(r, c)
			command.isEscaped = false
		}
	}

	if len(r) > 0 {
		result = append(result, string(r))
	}

	if !command.stdoutRedir && !command.stderrRedir {
		command.cmd = result[0]
		if len(result) > 1 {
			command.args = result[1:]
		}
		return
	}

	command.cmd = result[0]
	command.args = result[1 : len(result)-1]
	command.redirPath = result[len(result)-1]

}
