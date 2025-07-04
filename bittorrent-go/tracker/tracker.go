package tracker

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/lourencovales/codecrafters/bittorrent-go/bencode"
	"github.com/lourencovales/codecrafters/bittorrent-go/torrent"
)

// GetPeers contacts the tracker and retrieves a list of peers for the torrent.
func GetPeers(metaInfo *torrent.TorrentInfo, peerID [20]byte, port uint16) ([]string, error) {

	// Build the tracker URL with necessary query parameters
	trackerURL, err := buildTrackerUrl(metaInfo, peerID, port)
	if err != nil {
		return nil, err
	}

	// GET request to the tracker with the final query
	resp, err := http.Get(trackerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to contact tracker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK { // sanity check
		return nil, fmt.Errorf("tracker returned non-200 status: %s", resp.Status)
	}

	// parsing the answer
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read tracker response: %w", err)
	}

	// and unmarshal the bencoded response
	trackerResponse, err := bencode.Unmarshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracker response: %w", err)
	}

	return parsePeers(trackerResponse)
}

// buildTrackerURL constructs the full URL to query the tracker.
func buildTrackerUrl(metaInfo *torrent.TorrentInfo, peerID [20]byte, port uint16) (string, error) {

	base, err := url.Parse(metaInfo.AnnounceURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse announce URL: %w", err)
	}

	params := url.Values{
		"info_hash":  []string{string(metaInfo.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{strconv.Itoa(metaInfo.TotalLength)},
		"compact":    []string{"1"},
	}
	base.RawQuery = params.Encode()

	return base.String(), nil
}

// parsePeers extracts the peer list from the tracker's Bencoded response.
func parsePeers(trackerResponse interface{}) ([]string, error) {

	respDict, ok := trackerResponse.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid tracker response format")
	}

	peersValue, ok := respDict["peers"]
	if !ok {
		return nil, errors.New("tracker response missing 'peers' key")
	}

	peersString, ok := peersValue.(string)
	if !ok {
		return nil, errors.New("'peers' key is not a string")
	}

	peersBytes := []byte(peersString)
	if len(peersBytes)%6 != 0 {
		return nil, errors.New("malformed peers list")
	}

	var peerList []string
	for i := 0; i < len(peersBytes); i += 6 {
		ip := peersBytes[i : i+4]
		portBytes := peersBytes[i+4 : i+6]
		port := binary.BigEndian.Uint16(portBytes)
		peerList = append(peerList, fmt.Sprintf("%d.%d.%d.%d:%d", ip[0], ip[1], ip[2], ip[3], port))
	}

	return peerList, nil
}
