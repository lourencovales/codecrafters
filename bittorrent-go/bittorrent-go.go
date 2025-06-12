package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

// The decoder struct is a representation of the data structure that holds the
// all the information needed through the execution of the program
type decoder struct {
	command string
	args    string
	pos     int
}

func main() {
	// You can use print statements as follows for debugging, they'll be
	// visible when running tests.
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

		marsh, err := json.Marshal(decoded) // answer needs to be in json
		if err != nil {
			fmt.Println(fmt.Errorf("error marshalling json"))
		}

		fmt.Println(string(marsh))
	} else {
		fmt.Println("Unknown command: " + decoder.command)
		os.Exit(1)
	}
}

// The decodeBencode function is the main point of entry for data, where we
// select which type of bencoded data we will be dealing with, and dispatch it
// accordingly
func (decoder *decoder) decodeBencode() (interface{}, error) {

	if decoder.pos >= len(decoder.args) { // out-of-bounds check
		return nil, fmt.Errorf("unexpected end of input")
	}

	// the first character determines which type of data we're dealing with.
	// this will get updated as we progress through the input
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

// The decodeString function deals with bencoded strings
func (decoder *decoder) decodeString() (string, error) {

	var firstColonIndex int

	// we detect and save where the first colon is, as this is the delimiter
	// for bencoded strings
	for i := decoder.pos; i < len(decoder.args); i++ {
		if decoder.args[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	// the string we want to extract
	lengthStr := decoder.args[decoder.pos:firstColonIndex]
	// this is to extract the integer that codifies the string
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", err
	}

	// this is the full size of the string
	size := firstColonIndex + length + 1

	decodedString := decoder.args[firstColonIndex+1 : size]

	// the pos argument is updated with the full size of the string
	decoder.pos = size

	return decodedString, nil
}

// The decodeIntegers function deals with bencoded integers
func (decoder *decoder) decodeIntegers() (int, error) {

	// We start one char to the right to ignore the delimiter
	start := decoder.pos + 1
	end := start

	// we detect and save where the end of the integer is
	for ; end < len(decoder.args); end++ {
		if decoder.args[end] == 'e' {
			break
		}
	}

	if end >= len(decoder.args) {
		return 0, fmt.Errorf("malformed integer")
	}

	// the integer is between the start and end parameters
	result := decoder.args[start:end]

	// convert it to int type
	intResult, err := strconv.Atoi(result)
	if err != nil {
		return 0, err
	}

	// move the pos argument one to the right so that we ignore the last char
	decoder.pos = end + 1

	return intResult, nil
}

// The decodeList function deals with bencoded lists
func (decoder *decoder) decodeList() ([]interface{}, error) {

	result := make([]interface{}, 0)
	decoder.pos++ // ignoring the first char

	// while not out of bounds and we haven't found the char, keep going
	for decoder.pos < len(decoder.args) && decoder.args[decoder.pos] != 'e' {
		// recursively deal with each of the components of the list
		r, err := decoder.decodeBencode()
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}

	// move the pos argument one to the right so that we ignore the last char
	decoder.pos++
	return result, nil
}

// The decodeDict function deals with dictionaries
func (decoder *decoder) decodeDict() (map[string]interface{}, error) {

	result := make(map[string]interface{})
	decoder.pos++ // ignoring the first char

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

	// we're out of bounds or we never found the delimiter, so we error out
	if decoder.pos >= len(decoder.args) || decoder.args[decoder.pos] != 'e' {
		return nil, fmt.Errorf("malformed dictionary")
	}

	decoder.pos++ // Skip the delimiter
	return result, nil
}
