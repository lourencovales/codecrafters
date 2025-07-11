package tracker

import (
	"net/url"
	"testing"

	"github.com/lourencovales/codecrafters/bittorrent-go/torrent"
)

func TestBuildTrackerUrl(t *testing.T) {
	metaInfo := &torrent.TorrentInfo{
		AnnounceURL: "http://tracker.example.com:8080/announce",
		InfoHash:    [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		TotalLength: 1000000,
	}
	
	peerID := [20]byte{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}
	port := uint16(6881)

	result, err := buildTrackerUrl(metaInfo, peerID, port)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsedURL, err := url.Parse(result)
	if err != nil {
		t.Fatalf("failed to parse result URL: %v", err)
	}

	if parsedURL.Scheme != "http" {
		t.Errorf("expected scheme http, got %s", parsedURL.Scheme)
	}

	if parsedURL.Host != "tracker.example.com:8080" {
		t.Errorf("expected host tracker.example.com:8080, got %s", parsedURL.Host)
	}

	if parsedURL.Path != "/announce" {
		t.Errorf("expected path /announce, got %s", parsedURL.Path)
	}

	values := parsedURL.Query()
	
	if values.Get("port") != "6881" {
		t.Errorf("expected port 6881, got %s", values.Get("port"))
	}

	if values.Get("uploaded") != "0" {
		t.Errorf("expected uploaded 0, got %s", values.Get("uploaded"))
	}

	if values.Get("downloaded") != "0" {
		t.Errorf("expected downloaded 0, got %s", values.Get("downloaded"))
	}

	if values.Get("left") != "1000000" {
		t.Errorf("expected left 1000000, got %s", values.Get("left"))
	}

	if values.Get("compact") != "1" {
		t.Errorf("expected compact 1, got %s", values.Get("compact"))
	}

	if len(values.Get("info_hash")) != 20 {
		t.Errorf("expected info_hash length 20, got %d", len(values.Get("info_hash")))
	}

	if len(values.Get("peer_id")) != 20 {
		t.Errorf("expected peer_id length 20, got %d", len(values.Get("peer_id")))
	}
}

func TestBuildTrackerUrlInvalidURL(t *testing.T) {
	metaInfo := &torrent.TorrentInfo{
		AnnounceURL: "://invalid-url",
		InfoHash:    [20]byte{},
		TotalLength: 1000000,
	}
	
	peerID := [20]byte{}
	port := uint16(6881)

	_, err := buildTrackerUrl(metaInfo, peerID, port)
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestParsePeers(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
		hasError bool
	}{
		{
			name: "valid peers",
			input: map[string]interface{}{
				"peers": string([]byte{
					192, 168, 1, 1, 0x1A, 0xE1, // 192.168.1.1:6881
					10, 0, 0, 1, 0x1A, 0xE2,     // 10.0.0.1:6882
				}),
			},
			expected: []string{"192.168.1.1:6881", "10.0.0.1:6882"},
			hasError: false,
		},
		{
			name: "empty peers",
			input: map[string]interface{}{
				"peers": "",
			},
			expected: []string{},
			hasError: false,
		},
		{
			name:     "invalid response format",
			input:    "not a map",
			expected: nil,
			hasError: true,
		},
		{
			name:     "missing peers key",
			input:    map[string]interface{}{},
			expected: nil,
			hasError: true,
		},
		{
			name: "peers not a string",
			input: map[string]interface{}{
				"peers": 123,
			},
			expected: nil,
			hasError: true,
		},
		{
			name: "malformed peers list",
			input: map[string]interface{}{
				"peers": string([]byte{192, 168, 1, 1, 0x1A}), // Missing one byte
			},
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePeers(tt.input)
			
			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d peers, got %d", len(tt.expected), len(result))
			}

			for i, expectedPeer := range tt.expected {
				if i < len(result) && result[i] != expectedPeer {
					t.Errorf("at index %d: expected %s, got %s", i, expectedPeer, result[i])
				}
			}
		})
	}
}