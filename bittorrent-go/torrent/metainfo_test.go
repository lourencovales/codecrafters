package torrent

import (
	"os"
	"strings"
	"testing"

	"github.com/lourencovales/codecrafters/bittorrent-go/bencode"
)

func TestSplitPieceHashes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected int
		hasError bool
	}{
		{
			name:     "valid 40 bytes",
			input:    make([]byte, 40), // 2 pieces
			expected: 2,
			hasError: false,
		},
		{
			name:     "valid 20 bytes",
			input:    make([]byte, 20), // 1 piece
			expected: 1,
			hasError: false,
		},
		{
			name:     "empty",
			input:    []byte{},
			expected: 0,
			hasError: false,
		},
		{
			name:     "invalid length",
			input:    make([]byte, 19), // Not divisible by 20
			expected: 0,
			hasError: true,
		},
		{
			name:     "invalid length 21",
			input:    make([]byte, 21), // Not divisible by 20
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := splitPieceHashes(tt.input)
			
			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != tt.expected {
				t.Errorf("expected %d pieces, got %d", tt.expected, len(result))
			}

			for i, hash := range result {
				if len(hash) != 20 {
					t.Errorf("piece %d: expected hash length 20, got %d", i, len(hash))
				}
			}
		})
	}
}

func TestTorrentInfoString(t *testing.T) {
	info := &TorrentInfo{
		AnnounceURL: "http://tracker.example.com/announce",
		InfoHash:    [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		PieceHashes: [][20]byte{
			{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40},
			{41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60},
		},
		PieceLength: 262144,
		TotalLength: 1000000,
	}

	result := info.String()

	expectedStrings := []string{
		"Tracker URL: http://tracker.example.com/announce",
		"Length: 1000000",
		"Info Hash: 0102030405060708090a0b0c0d0e0f1011121314",
		"Piece Length: 262144",
		"Piece Hashes:",
		"15161718191a1b1c1d1e1f2021222324252627",
		"292a2b2c2d2e2f303132333435363738393a3b",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("expected string %q not found in result:\n%s", expected, result)
		}
	}
}

func TestHashInfoDict(t *testing.T) {
	// Create a simple bencode dict with info section
	infoDict := map[string]interface{}{
		"name":         "test.txt",
		"length":       1000,
		"piece length": 262144,
		"pieces":       "01234567890123456789", // 20 bytes
	}

	rootDict := map[string]interface{}{
		"announce": "http://tracker.example.com/announce",
		"info":     infoDict,
	}

	bencoded, err := bencode.Marshal(rootDict)
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}

	hash, err := hashInfoDict(bencoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash == [20]byte{} {
		t.Error("expected non-zero hash")
	}

	// Test consistency - same input should produce same hash
	hash2, err := hashInfoDict(bencoded)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if hash != hash2 {
		t.Error("hash function is not consistent")
	}
}

func TestHashInfoDictMissingInfo(t *testing.T) {
	// Create bencoded data without info dict
	rootDict := map[string]interface{}{
		"announce": "http://tracker.example.com/announce",
	}

	bencoded, err := bencode.Marshal(rootDict)
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}

	_, err = hashInfoDict(bencoded)
	if err == nil {
		t.Error("expected error for missing info dict")
	}
}

func TestParseFileNonExistent(t *testing.T) {
	_, err := ParseFile("nonexistent.torrent")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

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

func TestParseFileValid(t *testing.T) {
	tmpFile := createTestTorrentFile(t)
	defer os.Remove(tmpFile)

	info, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.AnnounceURL != "http://tracker.example.com/announce" {
		t.Errorf("expected announce URL 'http://tracker.example.com/announce', got '%s'", info.AnnounceURL)
	}

	if info.TotalLength != 1000 {
		t.Errorf("expected total length 1000, got %d", info.TotalLength)
	}

	if info.PieceLength != 262144 {
		t.Errorf("expected piece length 262144, got %d", info.PieceLength)
	}

	if len(info.PieceHashes) != 1 {
		t.Errorf("expected 1 piece hash, got %d", len(info.PieceHashes))
	}

	if info.InfoHash == [20]byte{} {
		t.Error("expected non-zero info hash")
	}
}

func TestParseFileInvalidBencode(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid.torrent")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte("invalid bencode")); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	_, err = ParseFile(tmpFile.Name())
	if err == nil {
		t.Error("expected error for invalid bencode")
	}
}