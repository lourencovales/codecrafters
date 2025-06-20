package main

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"
)

// TestDecodeString tests bencoded string decoding.
func TestDecodeString(t *testing.T) {
	d := &decoder{args: []byte("4:spam"), pos: 0}
	str, err := d.decodeString()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if str != "spam" {
		t.Errorf("expected 'spam', got '%s'", str)
	}
}

// TestDecodeInteger tests bencoded integer decoding.
func TestDecodeIntegers(t *testing.T) {
	d := &decoder{args: []byte("i42e"), pos: 0}
	val, err := d.decodeIntegers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

// TestDecodeList tests bencoded list decoding.
func TestDecodeList(t *testing.T) {
	d := &decoder{args: []byte("l4:spam4:eggse"), pos: 0}
	val, err := d.decodeList()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{"spam", "eggs"}
	if len(val) != 2 || val[0] != want[0] || val[1] != want[1] {
		t.Errorf("expected %v, got %v", want, val)
	}
}

// TestDecodeDict tests bencoded dictionary decoding.
func TestDecodeDict(t *testing.T) {
	d := &decoder{args: []byte("d3:cow3:moo4:spam4:eggse"), pos: 0}
	val, err := d.decodeDict()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val["cow"] != "moo" || val["spam"] != "eggs" {
		t.Errorf("unexpected map result: %v", val)
	}
}

// TestInfoHash tests the infoHash function on minimal input.
func TestInfoHash(t *testing.T) {
	// Create a dummy torrent with just "info" dictionary
	info := "d6:lengthi12345e12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaae"
	data := []byte("d8:announce13:tracker.com4:info" + info)

	// Write to a temp file
	tmp := "test.torrent"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp)

	d := &decoder{args: []byte(tmp)}
	hash, err := d.infoHash()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// hash is a [20]byte; just check it's not all zero
	if bytes.Equal(hash[:], make([]byte, 20)) {
		t.Errorf("infoHash returned all zeros")
	}
	t.Logf("infoHash: %s", hex.EncodeToString(hash[:]))
}
