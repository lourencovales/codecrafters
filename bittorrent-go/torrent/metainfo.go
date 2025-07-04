package torrent

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/lourencovales/codecrafters/bittorrent-go/bencode"
)

type TorrentInfo struct {
	AnnounceURL string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	TotalLength int
}

func (ti *TorrentInfo) String() string {

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tracker URL: %s\n", ti.AnnounceURL))
	sb.WriteString(fmt.Sprintf("Length: %d\n", ti.TotalLength))
	sb.WriteString(fmt.Sprintf("Info Hash: %x\n", ti.InfoHash))
	sb.WriteString(fmt.Sprintf("Piece Length: %d\n", ti.PieceLength))
	sb.WriteString("Piece Hashes:\n")
	for _, hash := range ti.PieceHashes {
		sb.WriteString(fmt.Sprintf("%s\n", hex.EncodeToString(hash[:])))
	}
	return sb.String()
}

func ParseFile(filename string) (*TorrentInfo, error) {

	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read torrent file: %w", err)
	}

	decodedData, err := bencode.Unmarshal(file)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bencoded file: %w", err)
	}

	metaDict, ok := decodedData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid torrent file format")
	}

	announceURL, ok := metaDict["announce"].(string)
	if !ok {
		return nil, fmt.Errorf("announce URL not found or invalid")
	}

	infoDict, ok := metaDict["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("info dict not found")
	}

	infoHash, err := hashInfoDict(file)
	if err != nil {
		return nil, err
	}

	pieceLength, ok := infoDict["piece length"].(int)
	if !ok {
		return nil, fmt.Errorf("piece len not found or invalid")
	}

	totalLength, ok := infoDict["length"].(int)
	if !ok {
		return nil, fmt.Errorf("file len not found or invalid")
	}

	piecesStr, ok := infoDict["pieces"].(string)
	if !ok {
		return nil, fmt.Errorf("pieces not found or invalid")
	}

	pieceHashes, err := splitPieceHashes([]byte(piecesStr))
	if err != nil {
		return nil, err
	}

	return &TorrentInfo{
		AnnounceURL: announceURL,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: pieceLength,
		TotalLength: totalLength,
	}, nil
}

func hashInfoDict(fileBytes []byte) ([20]byte, error) {

	infoStart := bytes.Index(fileBytes, []byte("4:info"))
	if infoStart == -1 {
		return [20]byte{}, fmt.Errorf("'info' dict not found")
	}

	infoSlice := fileBytes[infoStart+len("4:info"):]

	infoDecoded, err := bencode.Unmarshal(infoSlice)
	if err != nil {
		return [20]byte{}, fmt.Errorf("failed to decode info dict: %w", err)
	}

	infoBytes, err := bencode.Marshal(infoDecoded)
	if err != nil {
		return [20]byte{}, fmt.Errorf("failed to re-marshal info dict: %w", err)
	}

	return sha1.Sum(infoBytes), nil
}

func splitPieceHashes(pieces []byte) ([][20]byte, error) {

	if len(pieces)%20 != 0 {
		return nil, fmt.Errorf("malformed pieces string: length is not a multiple of 20")
	}
	numHashes := len(pieces) / 20
	hashes := make([][20]byte, numHashes)
	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], pieces[i*20:(i+1)*20])
	}
	return hashes, nil
}
