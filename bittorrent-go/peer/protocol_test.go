package peer

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestHasPiece(t *testing.T) {
	tests := []struct {
		name       string
		bitfield   []byte
		pieceIndex int
		expected   bool
	}{
		{"has piece 0", []byte{0x80}, 0, true},
		{"has piece 1", []byte{0x40}, 1, true},
		{"has piece 7", []byte{0x01}, 7, true},
		{"has piece 8", []byte{0x00, 0x80}, 8, true},
		{"does not have piece 0", []byte{0x7F}, 0, false},
		{"does not have piece 1", []byte{0xBF}, 1, false},
		{"index out of bounds", []byte{0xFF}, 8, false},
		{"empty bitfield", []byte{}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPiece(tt.bitfield, tt.pieceIndex)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatRequestPayload(t *testing.T) {
	tests := []struct {
		name     string
		index    uint32
		begin    uint32
		length   uint32
		expected []byte
	}{
		{"simple request", 0, 0, 16384, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x00}},
		{"piece 1 request", 1, 16384, 16384, []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x40, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatRequestPayload(tt.index, tt.begin, tt.length)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatRequestPayloadStructure(t *testing.T) {
	payload := FormatRequestPayload(123, 456, 789)
	
	if len(payload) != 12 {
		t.Errorf("expected payload length 12, got %d", len(payload))
	}
	
	index := binary.BigEndian.Uint32(payload[0:4])
	begin := binary.BigEndian.Uint32(payload[4:8])
	length := binary.BigEndian.Uint32(payload[8:12])
	
	if index != 123 {
		t.Errorf("expected index 123, got %d", index)
	}
	if begin != 456 {
		t.Errorf("expected begin 456, got %d", begin)
	}
	if length != 789 {
		t.Errorf("expected length 789, got %d", length)
	}
}

func TestMessageConstants(t *testing.T) {
	expectedConstants := map[string]uint8{
		"MsgChoke":         0,
		"MsgUnchoke":       1,
		"MsgInterested":    2,
		"MsgNotInterested": 3,
		"MsgHave":          4,
		"MsgBitfield":      5,
		"MsgRequest":       6,
		"MsgPiece":         7,
		"MsgCancel":        8,
	}

	actualConstants := map[string]uint8{
		"MsgChoke":         MsgChoke,
		"MsgUnchoke":       MsgUnchoke,
		"MsgInterested":    MsgInterested,
		"MsgNotInterested": MsgNotInterested,
		"MsgHave":          MsgHave,
		"MsgBitfield":      MsgBitfield,
		"MsgRequest":       MsgRequest,
		"MsgPiece":         MsgPiece,
		"MsgCancel":        MsgCancel,
	}

	for name, expected := range expectedConstants {
		if actual, ok := actualConstants[name]; !ok || actual != expected {
			t.Errorf("constant %s: expected %d, got %d", name, expected, actual)
		}
	}
}

func TestBlockSizeConstant(t *testing.T) {
	expectedBlockSize := 16 * 1024
	if BlockSize != expectedBlockSize {
		t.Errorf("expected BlockSize %d, got %d", expectedBlockSize, BlockSize)
	}
}

func TestMessageStruct(t *testing.T) {
	msg := Message{
		ID:      MsgPiece,
		Payload: []byte{0x01, 0x02, 0x03},
	}

	if msg.ID != MsgPiece {
		t.Errorf("expected ID %d, got %d", MsgPiece, msg.ID)
	}

	expectedPayload := []byte{0x01, 0x02, 0x03}
	if !bytes.Equal(msg.Payload, expectedPayload) {
		t.Errorf("expected payload %v, got %v", expectedPayload, msg.Payload)
	}
}