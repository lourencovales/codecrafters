package client

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/lourencovales/codecrafters/bittorrent-go/peer"
	"github.com/lourencovales/codecrafters/bittorrent-go/torrent"
	"github.com/lourencovales/codecrafters/bittorrent-go/tracker"
)

// Client is the struct that holds all the information needed for a single
// BitTorrent download session.
type Client struct {
	TorrentInfo *torrent.TorrentInfo
	Peers       []string
	PeerID      [20]byte
}

// New is the factory function that creates a new Client instance for any given
// torrent file.
func New(torrFile string) (*Client, error) {

	metaInfo, err := torrent.ParseFile(torrFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse torrent file: %w", err)
	}

	var peerID [20]byte
	if _, err := io.ReadFull(rand.Reader, peerID[:]); err != nil {
		return nil, fmt.Errorf("failed to generate peer ID: %w", err)
	}

	const listenPort uint16 = 6881 // TODO we might want to make this settable

	peers, err := tracker.GetPeers(metaInfo, peerID, listenPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get peers from tracker: %w", err)
	}

	return &Client{
		TorrentInfo: metaInfo,
		Peers:       peers,
		PeerID:      peerID,
	}, nil
}

// DownloadFile is the function that orchestrates the download of the file
func (c *Client) DownloadFile(outFile string) error {

	fileData := make([]byte, c.TorrentInfo.TotalLength)
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

// DownloadPiece is the exported function that orchestrates the download of a single
// piece and saves it to a file. It serves as a shim around the unexported
// downloadPiece function.
func (c *Client) DownloadPiece(outFile string, pieceIndex int) error {

	pieceData, err := c.downloadPiece(pieceIndex)
	if err != nil {
		return err
	}

	return os.WriteFile(outFile, pieceData, 0644)
}

// downloadPiece is an unexported function that contains the core logic for
// downloading a single piece by running through the list of available peers
func (c *Client) downloadPiece(pieceIndex int) ([]byte, error) {

	for _, peerAddr := range c.Peers {
		pieceData, err := c.tryDl(peerAddr, pieceIndex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to download from peer %s: %v. Trying next peer.\n", peerAddr, err)
			continue
		}

		expectedHash := c.TorrentInfo.PieceHashes[pieceIndex]
		actualHash := sha1.Sum(pieceData)
		if !bytes.Equal(expectedHash[:], actualHash[:]) {
			fmt.Fprintf(os.Stderr, "Piece hash mismatch for piece %d from peer %s. Trying next peer.\n", pieceIndex, peerAddr)
			continue
		}

		return pieceData, nil
	}

	return nil, errors.New("failed to download piece from any available peer")
}

// tryDL is an unexported function that contains the logic for downloading a
// piece from a single peer
func (c *Client) tryDl(peerAddr string, pieceIndex int) ([]byte, error) {

	conn, err := net.DialTimeout("tcp", peerAddr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = peer.Handshake(conn, c.TorrentInfo.InfoHash, c.PeerID)
	if err != nil {
		return nil, err
	}

	bitMsg, err := peer.ReadMsg(conn)
	if err != nil || bitMsg.ID != peer.MsgBitfield {
		return nil, errors.New("expected bitfield message")
	}

	if !peer.HasPiece(bitMsg.Payload, pieceIndex) {
		return nil, fmt.Errorf("peer does not have piece %d", pieceIndex)
	}

	if err = peer.SendMsg(conn, peer.MsgInterested, nil); err != nil {
		return nil, err
	}

	unchokeMsg, err := peer.ReadMsg(conn)
	if err != nil || unchokeMsg.ID != peer.MsgUnchoke {
		return nil, errors.New("unexpected unchoke message")
	}

	pieceSize := c.TorrentInfo.PieceLength
	if pieceIndex == len(c.TorrentInfo.PieceHashes)-1 {
		pieceSize = c.TorrentInfo.TotalLength % c.TorrentInfo.PieceLength
		if pieceSize == 0 {
			pieceSize = c.TorrentInfo.PieceLength
		}
	}

	pieceData := make([]byte, pieceSize)
	bytesDownloaded := 0
	blockSize := 16 * 1024

	for bytesDownloaded < pieceSize {
		length := blockSize
		if pieceSize-bytesDownloaded < length {
			length = pieceSize - bytesDownloaded
		}

		payload := peer.FormatRequestPayload(uint32(pieceIndex), uint32(bytesDownloaded), uint32(length))
		if err := peer.SendMsg(conn, peer.MsgRequest, payload); err != nil {
			return nil, err
		}

		pieceMsg, err := peer.ReadMsg(conn)
		if err != nil || pieceMsg.ID != peer.MsgPiece {
			return nil, errors.New("unexpected piece message")
		}

		blockData := pieceMsg.Payload[8:]
		copy(pieceData[bytesDownloaded:], blockData)
		bytesDownloaded += len(blockData)
	}

	return pieceData, nil
}
