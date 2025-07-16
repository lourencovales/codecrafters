package cmd

import (
	"os"
	"testing"

	"github.com/lourencovales/codecrafters/bittorrent-go/bencode"
)

func TestRunUnknownCommand(t *testing.T) {
	err := Run("unknown", []string{})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestRunDecodeInvalidArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"too many args", []string{"arg1", "arg2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run("decode", tt.args)
			if err == nil {
				t.Error("expected error for invalid args")
			}
		})
	}
}

func TestRunDecodeValid(t *testing.T) {
	// Test decoding a simple string
	err := Run("decode", []string{"4:spam"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunDecodeInvalidBencode(t *testing.T) {
	err := Run("decode", []string{"invalid"})
	if err == nil {
		t.Error("expected error for invalid bencode")
	}
}

func TestRunInfoInvalidArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"too many args", []string{"file1.torrent", "file2.torrent"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run("info", tt.args)
			if err == nil {
				t.Error("expected error for invalid args")
			}
		})
	}
}

func TestRunInfoNonExistentFile(t *testing.T) {
	err := Run("info", []string{"nonexistent.torrent"})
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
		"pieces":       "01234567890123456789", // 20 bytes
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

func TestRunInfoValid(t *testing.T) {
	tmpFile := createTestTorrentFile(t)
	defer os.Remove(tmpFile)

	err := Run("info", []string{tmpFile})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunPeersInvalidArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"too many args", []string{"file1.torrent", "file2.torrent"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run("peers", tt.args)
			if err == nil {
				t.Error("expected error for invalid args")
			}
		})
	}
}

func TestRunPeersValid(t *testing.T) {
	tmpFile := createTestTorrentFile(t)
	defer os.Remove(tmpFile)

	// This will fail because tracker doesn't exist, but that's expected
	err := Run("peers", []string{tmpFile})
	if err == nil {
		t.Error("expected error for non-existent tracker")
	}
}

func TestRunHandshakeInvalidArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"one arg", []string{"file.torrent"}},
		{"too many args", []string{"file.torrent", "peer1", "peer2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run("handshake", tt.args)
			if err == nil {
				t.Error("expected error for invalid args")
			}
		})
	}
}

func TestRunHandshakeValid(t *testing.T) {
	tmpFile := createTestTorrentFile(t)
	defer os.Remove(tmpFile)

	// This will fail because peer doesn't exist, but that's expected
	err := Run("handshake", []string{tmpFile, "192.168.1.1:6881"})
	if err == nil {
		t.Error("expected error for non-existent peer")
	}
}

func TestRunDownloadPieceInvalidArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"wrong flag", []string{"-x", "out.file", "file.torrent", "0"}},
		{"too few args", []string{"-o", "out.file", "file.torrent"}},
		{"too many args", []string{"-o", "out.file", "file.torrent", "0", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run("download_piece", tt.args)
			if err == nil {
				t.Error("expected error for invalid args")
			}
		})
	}
}

func TestRunDownloadPieceInvalidIndex(t *testing.T) {
	tmpFile := createTestTorrentFile(t)
	defer os.Remove(tmpFile)

	err := Run("download_piece", []string{"-o", "out.file", tmpFile, "invalid"})
	if err == nil {
		t.Error("expected error for invalid piece index")
	}
}

func TestRunDownloadPieceValid(t *testing.T) {
	tmpFile := createTestTorrentFile(t)
	defer os.Remove(tmpFile)

	outFile, err := os.CreateTemp("", "piece.out")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(outFile.Name())
	outFile.Close()

	// This will fail because tracker doesn't exist, but that's expected
	err = Run("download_piece", []string{"-o", outFile.Name(), tmpFile, "0"})
	if err == nil {
		t.Error("expected error for non-existent tracker")
	}
}

func TestRunDownloadInvalidArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"wrong flag", []string{"-x", "out.file", "file.torrent"}},
		{"too few args", []string{"-o", "out.file"}},
		{"too many args", []string{"-o", "out.file", "file.torrent", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run("download", tt.args)
			if err == nil {
				t.Error("expected error for invalid args")
			}
		})
	}
}

func TestRunDownloadValid(t *testing.T) {
	tmpFile := createTestTorrentFile(t)
	defer os.Remove(tmpFile)

	outFile, err := os.CreateTemp("", "download.out")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(outFile.Name())
	outFile.Close()

	// This will fail because tracker doesn't exist, but that's expected
	err = Run("download", []string{"-o", outFile.Name(), tmpFile})
	if err == nil {
		t.Error("expected error for non-existent tracker")
	}
}