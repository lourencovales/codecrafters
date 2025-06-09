package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

type Bs struct {
	command   string
	args      string
	firstChar rune
	lastChar  rune
	buffer    string
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	// TODO: this logic needs to get out of main
	bs := &Bs{}

	bs.command = os.Args[1]

	if bs.command == "decode" {

		bs.args = os.Args[2]
		bs.firstChar = rune(bs.args[0])

		decoded, err := bs.decodeBencode()
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(decoded)
	} else {
		fmt.Println("Unknown command: " + bs.command)
		os.Exit(1)
	}
}

func (bs *Bs) decodeBencode() (interface{}, error) {
	lenStr := len(bs.args)
	bs.lastChar = rune(bs.args[lenStr-1])
	if bs.isString() {
		unmarsh, err := bs.decodeString()
		if err != nil {
			return "", fmt.Errorf("error in string processing")
		}
		marsh, err := json.Marshal(unmarsh)
		if err != nil {
			return "", fmt.Errorf("error marshalling json")
		}
		return string(marsh), nil
	} else if bs.isInterger() {
		return bs.decodeIntegers()
	} else if bs.isList() {
		return bs.decodeList()
	} else {
		return "", fmt.Errorf("Only strings are supported at the moment")
	}
}

func (bs *Bs) decodeString() (string, error) {

	var firstColonIndex int

	for i := 0; i < len(bs.args); i++ {
		if bs.args[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := bs.args[:firstColonIndex]
	length, err := strconv.Atoi(lengthStr)
	decodedString := bs.args[firstColonIndex+1 : firstColonIndex+1+length]
	if err != nil {
		return "", err
	}
	if len(bs.buffer) >= length+firstColonIndex+1 {
		bs.buffer = bs.args[firstColonIndex+length+2:]
		bs.args = decodedString
		return bs.args, nil
	} else {
		bs.buffer = ""
	}

	return decodedString, nil
}

func (bs *Bs) decodeIntegers() (string, error) {

	var firstEIndex int

	for i := 0; i < len(bs.args); i++ {
		if bs.args[i] == 'e' {
			firstEIndex = i
			break
		}
	}

	result := bs.args[1:firstEIndex] // start at 1 since we know we can discard the 1st char

	if len(bs.buffer) >= len(result)+2 { // we need to add two to account for discarded chars
		bs.buffer = bs.args[firstEIndex+1:]
	} else {
		bs.buffer = ""
	}

	return result, nil
}

func (bs *Bs) decodeList() ([]string, error) {

	bsBuffer := &Bs{}
	bsBuffer.args = bs.args
	bsBuffer.firstChar = bs.firstChar
	result := []string{}

	// TODO: Add error checking
	bsBuffer.args, _ = strings.CutPrefix(bsBuffer.args, "l")
	bsBuffer.args, _ = strings.CutSuffix(bsBuffer.args, "e")
	bsBuffer.buffer = bsBuffer.args

	for bsBuffer.buffer != "" {
		if bsBuffer.isString() {
			bsBuffer.decodeString()
			//			bsBuffer.firstChar = rune(bsBuffer
			result = append(result, bsBuffer.args)
		}
		if bsBuffer.isInterger() {
			bsBuffer.decodeIntegers()
			result = append(result, bsBuffer.args)
		}
		if bsBuffer.isList() {
			bsBuffer.decodeList()
			result = append(result, bsBuffer.args)
		}
	}

	return result, nil
}

func (bs *Bs) isInterger() bool {
	return bs.firstChar == 'i'
}

func (bs *Bs) isString() bool {
	return unicode.IsDigit(bs.firstChar)
}

func (bs *Bs) isList() bool {
	return bs.firstChar == 'l'
}
