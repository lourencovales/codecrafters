package cmd

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/lourencovales/codecrafters/bittorrent-go/bencode"
	"github.com/lourencovales/codecrafters/bittorrent-go/client"
	"github.com/lourencovales/codecrafters/bittorrent-go/peer"
	"github.com/lourencovales/codecrafters/bittorrent-go/torrent"
	"github.com/lourencovales/codecrafters/bittorrent-go/tracker"
)

// The Run function is main point of entry for the CLI. It uses a switch
// statement to match the command to the first argument, and if no match is
// found it will fail and exit the software.
func Run(command string, args []string) error {

	switch command {
	case "decode":
		if len(args) != 1 {
			return errors.New("usage: decode <bencoded string>")
		}
		benValue := args[0]
		decoded, err := bencode.Unmarshal([]byte(benValue))
		if err != nil {
			return err
		}
		printJson(decoded)

	case "info":
		if len(args) != 1 {
			return errors.New("usage: info <torrent file>")
		}
		torrFile := args[0]
		metaInfo, err := torrent.ParseFile(torrFile)
		if err != nil {
			return err
		}
		fmt.Println(metaInfo.String())

	case "peers":
		if len(args) != 1 {
			return errors.New("usage: peers <torrent file>")
		}
		torrFile := args[0]
		metaInfo, err := torrent.ParseFile(torrFile)
		if err != nil {
			return err
		}

		var peerID [20]byte
		if _, err := io.ReadFull(rand.Reader, peerID[:]); err != nil {
			return fmt.Errorf("failed to generate peer ID: %w", err)
		}
		const listenPort uint16 = 6881

		peers, err := tracker.GetPeers(metaInfo, peerID, listenPort)
		if err != nil {
			return err
		}
		printJson(peers)

	case "handshake":
		if len(args) != 2 {
			return errors.New("usage: handshake <torrent file> <peer address>")
		}
		torrFile, peerAddr := args[0], args[1]
		metaInfo, err := torrent.ParseFile(torrFile)
		if err != nil {
			return err
		}

		var peerID [20]byte
		if _, err := io.ReadFull(rand.Reader, peerID[:]); err != nil {
			return fmt.Errorf("failed to generate peer ID: %w", err)
		}

		conn, err := net.DialTimeout("tcp", peerAddr, 3*time.Second)
		if err != nil {
			return fmt.Errorf("failed to connect to peer: %w", err)
		}
		defer conn.Close()

		recvPeerID, err := peer.Handshake(conn, metaInfo.InfoHash, peerID)
		if err != nil {
			return err
		}
		fmt.Printf("Peer ID: %x\n", recvPeerID)

	case "download_piece":
		if len(args) != 4 || args[0] != "-o" {
			return errors.New("usage: download_pieces -o <output file> <torrent file> <piece index>")
		}
		outFile, torrFile := args[1], args[2]
		pieceIndex, err := strconv.Atoi(args[3])
		if err != nil {
			return err
		}
		c, err := client.New(torrFile)
		if err != nil {
			return err
		}
		if err := c.DownloadPiece(outFile, pieceIndex); err != nil {
			return err
		}
		fmt.Printf("Piece %d downloaded to %s.\n", pieceIndex, outFile)

	case "download":
		if len(args) != 3 || args[0] != "-o" {
			return errors.New("usage: download -o <output file> <torrent file>")
		}
		outFile, torrFile := args[1], args[2]
		c, err := client.New(torrFile)
		if err != nil {
			return err
		}
		if err := c.DownloadFile(outFile); err != nil {
			return err
		}
		fmt.Printf("Downloaded %s to %s.\n", torrFile, outFile)

	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	return nil
}

// printJson is just a helper to format some output into JSON. It's unexported
// since it's only used for this package.
// TODO this is a remnant from the codecrafters challenge, need to revisit later
func printJson(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", " ")
	return encoder.Encode(v)
}
