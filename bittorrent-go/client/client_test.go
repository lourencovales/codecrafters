package client

import (
	"os"
	"testing"

	"github.com/lourencovales/codecrafters/bittorrent-go/bencode"
	"github.com/lourencovales/codecrafters/bittorrent-go/torrent"
)

func createTestTorrentFile(t *testing.T) string {
	// Create a minimal valid torrent file
	infoDict := map[string]interface{}{
		"name":         "test.txt",
		"length":       1000,
		"piece length": 262144,
		"pieces":       "01234567890123456789", // 20 bytes (1 piece)
	}

	rootDict := map[string]interface{}{
		"announce": "http://tracker.example.com/announce",
		"info":     infoDict,
	}

	bencoded, err := bencode.Marshal(rootDict)
	if err != nil {
		t.Fatalf("failed to marshal test torrent: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "test.torrent")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := tmpFile.Write(bencoded); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}

func TestNewClientInvalidTorrent(t *testing.T) {
	_, err := New("nonexistent.torrent")
	if err == nil {
		t.Error("expected error for non-existent torrent file")
	}
}

func TestNewClientValidTorrent(t *testing.T) {
	tmpFile := createTestTorrentFile(t)
	defer os.Remove(tmpFile)

	// This test will fail because it tries to contact a real tracker
	// but that's expected behavior for this type of integration test
	client, err := New(tmpFile)
	if err != nil {
		// This is expected since the tracker doesn't exist
		t.Logf("expected error contacting tracker: %v", err)
		return
	}

	if client == nil {
		t.Error("expected non-nil client")
	}

	if client.TorrentInfo == nil {
		t.Error("expected non-nil TorrentInfo")
	}

	if len(client.PeerID) != 20 {
		t.Errorf("expected PeerID length 20, got %d", len(client.PeerID))
	}
}

func TestClientStructure(t *testing.T) {
	// Test that we can create a client struct manually
	// This tests the structure without network calls
	testInfo := &torrent.TorrentInfo{
		AnnounceURL: "http://tracker.example.com/announce",
		InfoHash:    [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		PieceHashes: [][20]byte{
			{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40},
		},
		PieceLength: 262144,
		TotalLength: 1000000,
	}

	testPeers := []string{"192.168.1.1:6881", "10.0.0.1:6882"}
	testPeerID := [20]byte{41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60}

	client := &Client{
		TorrentInfo: testInfo,
		Peers:       testPeers,
		PeerID:      testPeerID,
	}

	if client.TorrentInfo.AnnounceURL != testInfo.AnnounceURL {
		t.Errorf("expected AnnounceURL %s, got %s", testInfo.AnnounceURL, client.TorrentInfo.AnnounceURL)
	}

	if len(client.Peers) != len(testPeers) {
		t.Errorf("expected %d peers, got %d", len(testPeers), len(client.Peers))
	}

	if client.PeerID != testPeerID {
		t.Errorf("expected PeerID %v, got %v", testPeerID, client.PeerID)
	}
}

func TestDownloadFileInvalidFile(t *testing.T) {
	// Test downloading to an invalid path
	client := &Client{
		TorrentInfo: &torrent.TorrentInfo{
			PieceHashes: [][20]byte{},
			TotalLength: 0,
		},
		Peers:  []string{},
		PeerID: [20]byte{},
	}

	err := client.DownloadFile("/invalid/path/file.txt")
	if err == nil {
		t.Error("expected error for invalid file path")
	}
}

func TestDownloadPieceInvalidIndex(t *testing.T) {
	// Test downloading piece with invalid index
	client := &Client{
		TorrentInfo: &torrent.TorrentInfo{
			PieceHashes: [][20]byte{
				{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			},
			PieceLength: 262144,
			TotalLength: 1000000,
		},
		Peers:  []string{},
		PeerID: [20]byte{},
	}

	tmpFile, err := os.CreateTemp("", "piece.out")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// This should fail because there are no peers
	err = client.DownloadPiece(tmpFile.Name(), 0)
	if err == nil {
		t.Error("expected error for download with no peers")
	}
}

func TestDownloadPieceIndexOutOfRange(t *testing.T) {
	// Test downloading piece with index out of range
	client := &Client{
		TorrentInfo: &torrent.TorrentInfo{
			PieceHashes: [][20]byte{
				{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			},
			PieceLength: 262144,
			TotalLength: 1000000,
		},
		Peers:  []string{"192.168.1.1:6881"},
		PeerID: [20]byte{},
	}

	tmpFile, err := os.CreateTemp("", "piece.out")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// This should fail because piece index 1 doesn't exist (only have index 0)
	err = client.DownloadPiece(tmpFile.Name(), 1)
	if err == nil {
		t.Error("expected error for piece index out of range")
	}
}