package main

import (
	"bytes"
	"crypto/rand"
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
// TODO: change the name
type decoder struct {
	command    string
	args       []byte
	pos        int
	peerAddr   string
	outputFile string
	pieceIndex int
}

// The PeerMessage struct is a rpresentation of the BitTorrent peer wire
// protocol message
type PeerMessage struct {
	ID      uint8
	Payload []byte
}

// TODO: description
type TorrentInfo struct {
	InfoHash    [20]byte
	PieceLength int
	PieceHashes [][20]byte
	TotalLength int
}

// These constants are useful for implementing the BT peer wire protocol
const (
	MsgChoke         = 0
	MsgUnchoke       = 1
	MsgInterested    = 2
	MsgNotInterested = 3
	MsgHave          = 4
	MsgBitfield      = 5
	MsgRequest       = 6
	MsgPiece         = 7
	MsgCancel        = 8
)

func main() {
	cmd := os.Args[1]
	d := &decoder{command: cmd}

	var result interface{}
	var err error

	switch cmd {
	case "decode":
		d.pos = 0
		d.args = []byte(os.Args[2])
		result, err = d.decodeBencode()
	case "info":
		d.args = []byte(os.Args[2])
		result, err = d.info()
	case "peers":
		d.args = []byte(os.Args[2])
		result, err = d.peers()
	case "handshake":
		d.args = []byte(os.Args[2])
		d.peerAddr = os.Args[3]
		result, err = d.handshake()
	case "download_file":
		d.outputFile = os.Args[3]
		d.args = os.Args[4]
		d.pieceIndex = os.Args[5]
		result, err = d.download()
	default:
		fmt.Fprintln(os.Stderr, "Unknown command:", cmd)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// it's expected that these two commands are printed as string, not JSON
	if cmd == "handshake" || cmd == "info" || cmd == "download_file" {
		if str, ok := result.(string); ok {
			fmt.Println(str)
			return
		}
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

	// getting the infoHash
	infoHash, err := d.infoHash()
	hashHex := hex.EncodeToString(infoHash[:])

	// for URL, Length and Pieces params, make use of the already existing
	// functions and parse the results with a bit of type assertion magic

	// file ingestion
	file, err := os.ReadFile(string(d.args))
	if err != nil {
		return nil, err
	}
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

	infoHash, err := d.infoHash()
	if err != nil {
		return nil, err
	}

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

func (d *decoder) handshake() (interface{}, error) {

	// getting the infoHash
	infoHash, err := d.infoHash()
	if err != nil {
		return nil, err
	}

	// 20 byte random peer id
	peerID := make([]byte, 20)
	if _, err = io.ReadFull(rand.Reader, peerID); err != nil {
		return nil, fmt.Errorf("failed to generate peer id: %w", err)
	}

	// build the handshake msg
	handshake := new(bytes.Buffer)
	handshake.WriteByte(19)
	handshake.WriteString("BitTorrent protocol")
	handshake.Write(make([]byte, 8)) // 8 zero bytes
	handshake.Write(infoHash[:])
	handshake.Write(peerID)

	// connect to peer and send handshake
	conn, err := net.Dial("tcp", d.peerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write(handshake.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to send handshake: %w", err)
	}

	handshakeResponse := make([]byte, 68) // 68 bytes is the default response
	if _, err := io.ReadFull(conn, handshakeResponse); err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %w", err)
	}

	if handshakeResponse[0] != 19 || string(handshakeResponse[1:20]) != "BitTorrent protocol" {
		return nil, fmt.Errorf("invalid handshake response")
	}

	recvPeerID := handshakeResponse[48:68]

	return fmt.Sprintf("Peer ID: %x", recvPeerID), nil
}

func (d *decoder) download() (interface{}, error) {
	torrentInfo, err := d.getTorrentInfo()
	if err != nil {
		return nil, fmt.Errorf("problem with torrent parsing: %w", err)
	}

	peers, err := d.peers()
	if err != nil {
		return nil, fmt.Errorf("error getting the peer list: %w", err)
	}

	peerList, ok := peers.([]string)
	if !ok || len(peerList) == 0 {
		return nil, fmt.Errorf("no peers available")
	}

	for _, peer := range peerList {
		d.peerAddr = peer
		pieceData, err := d.downloadFromPeer(torrentInfo, d.pieceIndex)
		if err != nil {
			continue
		}

		expectedHash := torrentInfo.PieceHashes[pieceIndex]
		actualHash := sha1.Sum(pieceData)

		if !bytes.Equal(expectedHash[:], actualHash[:]) {
			continue
		}

		if err := os.WriteFile(d.outputFile, pieceData, 0644); err != nil {
			return nil, fmt.Errorf("failed to write data to file: %w", err)
		}

		return fmt.Sprintf("Piece %d downloaded to %s", pieceIndex, d.outputFile), nil
	}

	return nil, fmt.Errorf("unable to download piece from any peer")

}

func (d *decoder) getTorrentInfo() (*TorrentInfo, error) {
	file, err := os.ReadFile(string(d.args))
	if err != nil {
		return nil, err
	}

	decoder := &decoder{args: file}
	decoded, err := decoder.decodeDict()
	if err != nil {
		return nil, err
	}

	infoMap, ok := decoded["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid info section")
	}

	pieceLength, ok := infoMap["piece length"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid piece length")
	}

	totalLength, ok := infoMap["length"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid file length")
	}

	piecesStr, ok := infoMap["pieces"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid pieces")
	}

	pieces := []byte(piecesStr)
	var pieceHashes [][20]byte
	for i := 0; i < len(pieces); i += 20 {
		var hash [20]byte
		copy(hash[:], pieces[i:i+20])
		pieceHashes = append(pieceHashes, hash)
	}

	infoHash, err := d.infoHash()
	if err != nil {
		return nil, err
	}

	return &TorrentInfo{
		InfoHash:    infoHash,
		PieceLength: pieceLength,
		PieceHashes: pieceHashes,
		TotalLength: totalLength,
	}, nil
}

func (d *decoder) downloadFromPeer(tInfo *TorrentInfo, pIndex int) ([]byte, error) {
	panic()
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
	encoder.SetIndent("", "")
	return encoder.Encode(v)
}

func (d *decoder) infoHash() ([20]uint8, error) {
	var empty [20]uint8

	// ingest the file
	file, err := os.ReadFile(string(d.args))
	if err != nil {
		return empty, err
	}

	// need to figure out where the info dict is
	start := bytes.Index(file, []byte("4:info"))
	if start == -1 {
		return empty, fmt.Errorf("info dict not found")
	}
	start += len("4:info")

	// decode just that data and calculate the hash
	subDec := &decoder{args: file, pos: start}
	startPos := subDec.pos
	_, err = subDec.decodeDict()
	if err != nil {
		return empty, err
	}
	endPos := subDec.pos

	return sha1.Sum(file[startPos:endPos]), nil
}
