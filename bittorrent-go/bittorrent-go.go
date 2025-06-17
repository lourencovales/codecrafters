package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"unicode"
)

// The decoder struct is a representation of the data structure that holds the
// all the information needed through the execution of the program
type decoder struct {
	command string
	args    []byte
	pos     int
}

func main() {
	cmd, arg := os.Args[1], os.Args[2]
	d := &decoder{command: cmd, args: []byte(arg)}

	var result interface{}
	var err error

	switch cmd {
	case "decode":
		d.pos = 0
		result, err = d.decodeBencode()
	case "info":
		result, err = d.info()
	case "peers":
		result, err = d.peers()
	default:
		fmt.Fprintln(os.Stderr, "Unknown command:", cmd)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	printJson(result)
}

// The decodeBencode function is the main point of entry for data, where we
// select which type of bencoded data we will be dealing with, and dispatch it
// accordingly
func (d *decoder) decodeBencode() (interface{}, error) {

	if d.pos >= len(d.args) { // out-of-bounds check
		return nil, fmt.Errorf("unexpected end of input")
	}

	// the first character determines which type of data we're dealing with.
	// this will get updated as we progress through the input
	firstChar := rune(d.args[d.pos])
	if unicode.IsDigit(firstChar) {
		return d.decodeString()
	}
	if firstChar == 'i' {
		return d.decodeIntegers()
	}
	if firstChar == 'l' {
		return d.decodeList()
	}
	if firstChar == 'd' {
		return d.decodeDict()
	}
	return "", fmt.Errorf("unsupported bencode: %d, %c", d.pos, firstChar)
}

// The info function serves as a way to leverage the decodeBencode function to
// extract information from torrent files
func (d *decoder) info() (interface{}, error) {

	// file ingestion
	file, err := os.ReadFile(string(d.args))
	if err != nil {
		return nil, err
	}

	// need to figure out where the info dict is
	start := bytes.Index(file, []byte("4:info"))
	if start == -1 {
		return nil, fmt.Errorf("info dict not found")
	}
	start += len("4:info")

	// decode just that data and calculate the hash
	subDec := &decoder{args: file, pos: start}
	_, err = subDec.decodeDict()
	if err != nil {
		return nil, err
	}
	end := subDec.pos

	infoHash := sha1.Sum(file[start:end])
	hashHex := hex.EncodeToString(infoHash[:])

	// for URL, Length and Pieces params, make use of the already existing
	// functions and parse the results with a bit of type assertion magic
	d.args = file

	decoded, err := d.decodeDict()
	if err != nil {
		return nil, err
	}

	subMap, ok := decoded["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("problem with type assertion of info field")
	}

	sPieces, ok := subMap["pieces"].(string)
	if !ok {
		return nil, fmt.Errorf("problem with type assertion of pieces field")
	}
	pieces := []byte(sPieces)

	var hashes []string
	for i := 0; i < len(pieces); i += 20 { // assuming 20byte chunks
		end := i + 20
		if end > len(pieces) { // bounds checking
			end = len(pieces)
		}
		hashes = append(hashes, hex.EncodeToString(pieces[i:end]))
	}

	return fmt.Sprintf(
		"Tracker URL: %s\nLength: %d\nInfo Hash: %s\nPiece Length: %d\nPieces: %s\n",
		decoded["announce"],
		subMap["length"],
		hashHex,
		subMap["piece length"],
		hashes,
	), nil
}

// The peers function is responsible for interacting with the bittorrent tracker
// and getting information from it - namely, peers
func (d *decoder) peers() (interface{}, error) {

	const (
		peerId     = "thisisa20charsstring"
		port       = 6881
		uploaded   = 0
		downloaded = 0
		compact    = 1
	)

	// file ingestion
	file, err := os.ReadFile(string(d.args))
	if err != nil {
		return nil, err
	}

	decoded, err := (&decoder{args: file}).decodeDict()
	if err != nil {
		return nil, err
	}

	urlAnnounce, ok := decoded["announce"].(string)
	if !ok {
		return nil, fmt.Errorf("announce field is not a string")
	}

	// need to figure out where the info dict is
	start := bytes.Index(file, []byte("4:info"))
	if start == -1 {
		return nil, fmt.Errorf("info dict not found")
	}
	start += len("4:info")

	// decode just that data and calculate the hash
	subDec := &decoder{args: file, pos: start}
	_, err = subDec.decodeDict()
	if err != nil {
		return nil, err
	}

	infoHash := sha1.Sum(file[start:subDec.pos])

	subMap, ok := decoded["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("problem with type assertion of info field")
	}

	left, ok := subMap["length"].(int)
	if !ok {
		return nil, fmt.Errorf("problem with type assertion of length field")
	}

	// building the query to communicate with the tracker
	queries := url.Values{}
	queries.Add("info_hash", string(infoHash[:]))
	queries.Add("peer_id", peerId)
	queries.Add("port", strconv.Itoa(port))
	queries.Add("uploaded", strconv.Itoa(uploaded))
	queries.Add("downloaded", strconv.Itoa(downloaded))
	queries.Add("left", strconv.Itoa(left))
	queries.Add("compact", strconv.Itoa(compact))
	query := queries.Encode()

	finalUrl := urlAnnounce + "?" + query

	// GET request to the tracker with the final query
	resp, err := http.Get(finalUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// we parse the answer
	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error getting 200 answer from the tracker")
	}
	if err != nil {
		return nil, err
	}

	respDecoder := &decoder{args: body}
	response, err := respDecoder.decodeDict()
	if err != nil {
		return nil, err
	}

	peersResp, ok := response["peers"].(string)
	if !ok {
		return nil, fmt.Errorf("problem with type assertion of peers field")
	}

	peersData := []byte(peersResp)
	if len(peersData)%6 != 0 { // sanity checking
		return nil, fmt.Errorf("peer data is malformed")
	}

	// we extract the peer information from the answer
	var ipList []string
	for i := 0; i < len(peersData); i += 6 {
		ip := net.IPv4(
			peersData[i],
			peersData[i+1],
			peersData[i+2],
			peersData[i+3],
		)
		port := binary.BigEndian.Uint16(peersData[i+4 : i+6])
		ipList = append(ipList, fmt.Sprintf("%s:%d", ip.String(), port))
	}

	return ipList, nil
}

// The decodeString function deals with bencoded strings
func (d *decoder) decodeString() (string, error) {

	var firstColonIndex int

	// we detect and save where the first colon is, as this is the delimiter
	// for bencoded strings
	for i := d.pos; i < len(d.args); i++ {
		if d.args[i] == ':' {
			firstColonIndex = i
			break
		}
	}
	if firstColonIndex == 0 || firstColonIndex <= d.pos { // sanity checking
		return "", fmt.Errorf("problems with string parsing")
	}

	// the string we want to extract
	lengthStr := d.args[d.pos:firstColonIndex]
	// this is to extract the integer that codifies the string
	length, err := strconv.Atoi(string(lengthStr))
	if err != nil {
		return "", err
	}

	// this is the full size of the string
	size := firstColonIndex + length + 1

	decodedString := d.args[firstColonIndex+1 : size]

	// the pos argument is updated with the full size of the string
	d.pos = size

	return string(decodedString), nil
}

// The decodeIntegers function deals with bencoded integers
func (d *decoder) decodeIntegers() (int, error) {

	// We start one char to the right to ignore the delimiter
	start := d.pos + 1
	end := start

	// we detect and save where the end of the integer is
	for ; end < len(d.args); end++ {
		if d.args[end] == 'e' {
			break
		}
	}

	if end >= len(d.args) {
		return 0, fmt.Errorf("malformed integer")
	}

	// the integer is between the start and end parameters
	result := d.args[start:end]

	// convert it to int type
	intResult, err := strconv.Atoi(string(result))
	if err != nil {
		return 0, fmt.Errorf("problem with int parsing: %w", err)
	}

	// move the pos argument one to the right so that we ignore the last char
	d.pos = end + 1

	return intResult, nil
}

// The decodeList function deals with bencoded lists
func (d *decoder) decodeList() ([]interface{}, error) {

	result := make([]interface{}, 0)
	d.pos++ // ignoring the first char

	// while not out of bounds and we haven't found the char, keep going
	for d.pos < len(d.args) && d.args[d.pos] != 'e' {
		// recursively deal with each of the components of the list
		r, err := d.decodeBencode()
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}

	if d.pos >= len(d.args) { // sanity checking
		return nil, fmt.Errorf("list is malformed")
	}

	// move the pos argument one to the right so that we ignore the last char
	d.pos++
	return result, nil
}

// The decodeDict function deals with dictionaries
func (d *decoder) decodeDict() (map[string]interface{}, error) {

	result := make(map[string]interface{})
	d.pos++ // ignoring the first char

	var lastKey string

	for d.pos < len(d.args) && d.args[d.pos] != 'e' {
		// Keys must be strings
		keyRaw, err := d.decodeString()
		if err != nil {
			return nil, fmt.Errorf("invalid dictionary key: %w", err)
		}

		// Ensure lexicographical key order
		if lastKey != "" && keyRaw < lastKey {
			return nil, fmt.Errorf("dictionary keys not in lex order: %q < %q", keyRaw, lastKey)
		}
		lastKey = keyRaw

		value, err := d.decodeBencode()
		if err != nil {
			return nil, fmt.Errorf("invalid dictionary value for key %q: %w", keyRaw, err)
		}

		result[keyRaw] = value
	}

	// we're out of bounds or we never found the delimiter, so we error out
	if d.pos >= len(d.args) || d.args[d.pos] != 'e' {
		return nil, fmt.Errorf("malformed dictionary")
	}

	d.pos++ // Skip the delimiter
	return result, nil
}

func printJson(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
