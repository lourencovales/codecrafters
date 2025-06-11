package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

type decoder struct {
	command string
	args    string
	pos     int
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	// TODO: this logic needs to get out of main, probably
	decoder := &decoder{}

	decoder.command = os.Args[1]

	if decoder.command == "decode" {

		decoder.args = os.Args[2]
		decoder.pos = 0

		decoded, err := decoder.decodeBencode()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		marsh, err := json.Marshal(decoded)
		if err != nil {
			fmt.Println(fmt.Errorf("error marshalling json"))
		}

		fmt.Println(string(marsh))
	} else {
		fmt.Println("Unknown command: " + decoder.command)
		os.Exit(1)
	}
}

func (decoder *decoder) decodeBencode() (interface{}, error) {

	if decoder.pos >= len(decoder.args) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	firstChar := rune(decoder.args[decoder.pos])
	if unicode.IsDigit(firstChar) {
		return decoder.decodeString()
	}
	if firstChar == 'i' {
		return decoder.decodeIntegers()
	}
	if firstChar == 'l' {
		return decoder.decodeList()
	}
	if firstChar == 'd' {
		return decoder.decodeDict()
	}
	return "", fmt.Errorf("unsupported bencode: %d, %c", decoder.pos, firstChar)
}

func (decoder *decoder) decodeString() (string, error) {

	var firstColonIndex int

	for i := decoder.pos; i < len(decoder.args); i++ {
		if decoder.args[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := decoder.args[decoder.pos:firstColonIndex]
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", err
	}

	size := firstColonIndex + length + 1

	decodedString := decoder.args[firstColonIndex+1 : size]

	decoder.pos = size

	return decodedString, nil
}

func (decoder *decoder) decodeIntegers() (int, error) {

	start := decoder.pos + 1
	end := start

	for ; end < len(decoder.args); end++ {
		if decoder.args[end] == 'e' {
			break
		}
	}

	if end >= len(decoder.args) {
		return 0, fmt.Errorf("malformed integer")
	}

	result := decoder.args[start:end]

	intResult, err := strconv.Atoi(result)
	if err != nil {
		return 0, err
	}

	decoder.pos = end + 1

	return intResult, nil
}

func (decoder *decoder) decodeList() ([]interface{}, error) {

	result := make([]interface{}, 0)
	decoder.pos++

	for decoder.pos < len(decoder.args) && decoder.args[decoder.pos] != 'e' {
		r, err := decoder.decodeBencode()
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}

	decoder.pos++
	return result, nil
}

func (decoder *decoder) decodeDict() (map[string]interface{}, error) {

	result := make(map[string]interface{})
	decoder.pos++ // Skip the 'd'

	var lastKey string

	for decoder.pos < len(decoder.args) && decoder.args[decoder.pos] != 'e' {
		// Keys must be strings
		keyRaw, err := decoder.decodeString()
		if err != nil {
			return nil, fmt.Errorf("invalid dictionary key: %w", err)
		}

		// Ensure lexicographical key order
		if lastKey != "" && keyRaw < lastKey {
			return nil, fmt.Errorf("dictionary keys not in lex order: %q < %q", keyRaw, lastKey)
		}
		lastKey = keyRaw

		value, err := decoder.decodeBencode()
		if err != nil {
			return nil, fmt.Errorf("invalid dictionary value for key %q: %w", keyRaw, err)
		}

		result[keyRaw] = value
	}

	if decoder.pos >= len(decoder.args) || decoder.args[decoder.pos] != 'e' {
		return nil, fmt.Errorf("unterminated dictionary")
	}

	decoder.pos++ // Skip the 'e'
	return result, nil
}
