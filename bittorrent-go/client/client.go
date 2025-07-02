package client

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

type Client struct {
	TorrentInfo *torrent.TorrentInfo
	Peers       []string
	PeerID      [20]byte
}

func New(torrFile string) (*Client, error) {

	metaInfo, err := torrent.ParseFile(torrFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse torrent file: %w", err)
	}

	peers, err := tracker.GetPeers(metaInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get peers from tracker: %w", err)
	}

	var peerID [20]byte
	if _, err := io.ReadFull(rand.Reader, peerID[:]); err != nil {
		return nil, fmt.Errorf("failed to generate peer ID: %w", err)
	}

	return &Client{
		TorrentInfo: metaInfo,
		Peers:       peers,
		PeerID:      peerID,
	}, nil
}

func (c *Client) DownloadFile(outFile string) error {

	fileData := make([]byte, c.TorrentFile.TotalLength)
	pieceCount := len(c.TorrentInfo.PieceHashes)

	for i := 0; i < pieceCount; i++ {
		fmt.Printf("Downloading piece %d of %d...\n", i+1, pieceCount)
		pieceData, err := c.downloadPiece(i)
		if err != nil {
			return fmt.Errorf("failed to download piece %d: %w", i, err)
		}
		start := i * c.TorrentInfo.PieceLength
		copy(fileData[start:], pieceData)
	}

	return os.WriteFile(outFile, fileData, 0644)
}

func (c *Client) DownloadPiece(outFile string, pieceIndex int) error {

	pieceData, err := c.downlaodPiece(pieceIndex)
	if err != nil {
		return err
	}

	return os.WriteFile(outFile, pieceData, 0644)
}
