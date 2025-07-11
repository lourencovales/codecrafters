package peer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// These are constants for the peer wire protocol.
const (
	MsgChoke         uint8 = 0
	MsgUnchoke       uint8 = 1
	MsgInterested    uint8 = 2
	MsgNotInterested uint8 = 3
	MsgHave          uint8 = 4
	MsgBitfield      uint8 = 5
	MsgRequest       uint8 = 6
	MsgPiece         uint8 = 7
	MsgCancel        uint8 = 8
)

// Keeping block sizes constant at 16KiB
// TODO might want to make this settable
const BlockSize = 16 * 1024

// Message struct is a representation of the parts of a message in the BT peer
// wire protocol.
type Message struct {
	ID      uint8
	Payload []byte
}

// Handshake performs the BT handshake with a peer.
func Handshake(conn net.Conn, infoHash [20]byte, peerID [20]byte) ([20]byte, error) {

	// Build a new handshake message
	handshake := new(bytes.Buffer)
	handshake.WriteByte(19)
	handshake.WriteString("BitTorrent protocol")
	handshake.Write(make([]byte, 8))
	handshake.Write(infoHash[:])
	handshake.Write(peerID[:])

	if _, err := conn.Write(handshake.Bytes()); err != nil {
		return [20]byte{}, err
	}

	response := make([]byte, 68)
	if _, err := io.ReadFull(conn, response); err != nil {
		return [20]byte{}, err
	}

	if response[0] != 19 || string(response[1:20]) != "BitTorrent protocol" {
		return [20]byte{}, fmt.Errorf("invalid handshake response")
	}

	var receivedPeerID [20]byte
	copy(receivedPeerID[:], response[48:68])
	return receivedPeerID, nil
}

// SendMsg is a function that serializes and send a message to a peer.
func SendMsg(conn net.Conn, msgID uint8, payload []byte) error {
	var buf bytes.Buffer
	msgLen := uint32(1 + len(payload))

	if err := binary.Write(&buf, binary.BigEndian, msgLen); err != nil {
		return err
	}
	buf.WriteByte(msgID)
	if payload != nil {
		buf.Write(payload)
	}

	_, err := conn.Write(buf.Bytes())
	return err
}

// ReadMsg is a function that reads a peer wire protocol message
func ReadMsg(conn net.Conn) (*Message, error) {

	var msgLen uint32
	if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
		return nil, err
	}

	if msgLen == 0 {
		// Keep-alive message
		return &Message{}, nil
	}

	payload := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	return &Message{
		ID:      payload[0],
		Payload: payload[1:],
	}, nil
}

// HasPiece checks if a peer has a specific piece base on bitfield
func HasPiece(bitfield []byte, pieceIndex int) bool {
	byteIndex := pieceIndex / 8
	bitIndex := pieceIndex % 8

	if byteIndex >= len(bitfield) {
		return false
	}

	return (bitfield[byteIndex] & (1 << (7 - bitIndex))) != 0
}

// FormatRequestPayload creates the payload for a 'request' message.
func FormatRequestPayload(index, begin, length uint32) []byte {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], index)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)
	return payload
}
