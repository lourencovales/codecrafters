package bencode

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"unicode"
)

// decoder is the data structure that will hold the data and position of the
// reader throughout the package
type decoder struct {
	data []byte
	pos  int
}

// Unmarshal is the entry point for the bencode package, it accepts bencoded
// data and returns it as a Go interface. It's the exported counterpart of the
// decode function.
// TODO the interface{} part is also a remnant of the codecrafters challenge,
// might want to revisit this as it complicates things with type assertions
func Unmarshal(data []byte) (interface{}, error) {

	if len(data) == 0 {
		return nil, fmt.Errorf("bencode: cannot unmarshal empty data")
	}
	d := &decoder{data: data, pos: 0}
	return d.decode()
}

// decode is the unexported function that matches the bencoded data with its
// type and dispatches it accordingly
func (d *decoder) decode() (interface{}, error) {

	if d.pos >= len(d.data) {
		return nil, fmt.Errorf("bencode: unexpected end of input")
	}

	switch d.data[d.pos] {
	case 'i':
		return d.decodeInt()
	case 'l':
		return d.decodeList()
	case 'd':
		return d.decodeDict()
	default:
		if unicode.IsDigit(rune(d.data[d.pos])) {
			return d.decodeString()
		}
		return nil, fmt.Errorf("bencode: invalid character '%c' at position %d", d.data[d.pos], d.pos)
	}
}

// decodeString function deals with bencoded strings
func (d *decoder) decodeString() (string, error) {

	// we detect and save where the first colon is, as this is the delimiter
	// for bencoded strings
	colonIndex := bytes.IndexByte(d.data[d.pos:], ':')
	if colonIndex == -1 {
		return "", fmt.Errorf("bencode: invalid string format, missing colon")
	}
	colonIndex += d.pos

	// the string we want to extract
	lengthStr := string(d.data[d.pos:colonIndex])
	// this is to extract the integer that codifies the string
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", fmt.Errorf("bencode: invalid string length '%s'", lengthStr)
	}

	start := colonIndex + 1
	end := start + length
	if end > len(d.data) {
		return "", fmt.Errorf("bencode: string length exceeds data boundary")
	}

	// the pos argument is updated with the full size of the string
	d.pos = end
	return string(d.data[start:end]), nil
}

// decodeInt function deals with bencoded integers
func (d *decoder) decodeInt() (int, error) {
	d.pos++                                          // Skip 'i'
	endIndex := bytes.IndexByte(d.data[d.pos:], 'e') // find the delimiter
	if endIndex == -1 {
		return 0, fmt.Errorf("bencode: invalid integer format, missing 'e'")
	}
	endIndex += d.pos // the integer is between the start and end parameters

	intStr := string(d.data[d.pos:endIndex])
	val, err := strconv.Atoi(intStr)
	if err != nil {
		return 0, fmt.Errorf("bencode: invalid integer value '%s'", intStr)
	}

	// move the pos argument one to the right so that we ignore the last char
	d.pos = endIndex + 1
	return val, nil
}

// decodeList function deals with bencoded lists
func (d *decoder) decodeList() ([]interface{}, error) {
	d.pos++ // Skip 'l'
	var list []interface{}

	// while not out of bounds and we haven't found the char, keep going
	for d.pos < len(d.data) && d.data[d.pos] != 'e' {
		val, err := d.decode()
		if err != nil {
			return nil, err
		}
		list = append(list, val)
	}

	if d.pos >= len(d.data) || d.data[d.pos] != 'e' {
		return nil, fmt.Errorf("bencode: invalid list format, missing 'e'")
	}

	d.pos++ // Skip 'e'
	return list, nil
}

// decodeDict function deals with bencoded dictionaries
func (d *decoder) decodeDict() (map[string]interface{}, error) {
	d.pos++ // Skip 'd'
	dict := make(map[string]interface{})

	// var lastKey string // not needed if we're not checking for key ordering

	for d.pos < len(d.data) && d.data[d.pos] != 'e' {
		key, err := d.decodeString()
		if err != nil {
			return nil, err
		}

		// The BitTorrent spec wants us to ensure lexicographical key order, but
		// this creates more problems than it solves. Need to find a better way
		// to deal with this

		//if lastKey != "" && keyRaw < lastKey {
		//	fmt.Println(fmt.Errorf("dictionary keys not in lex order: %q < %q", keyRaw, lastKey))
		//}
		//lastKey = keyRaw

		val, err := d.decode()
		if err != nil {
			return nil, err
		}
		dict[key] = val
	}

	// we're out of bounds or we never found the delimiter, so we error out
	if d.pos >= len(d.data) || d.data[d.pos] != 'e' {
		return nil, fmt.Errorf("bencode: invalid dictionary format, missing 'e'")
	}

	d.pos++ // Skip 'e'
	return dict, nil
}

// Marshal converts a Go interface into Bencoded data - this is the exported
// shim around marshalTo
func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := marshalTo(&buf, v)
	return buf.Bytes(), err
}

// marshalTo does the heavy lifting of marshaling the data depending on the type
// assertion match
func marshalTo(buf *bytes.Buffer, v interface{}) error {
	switch val := v.(type) {
	case string:
		fmt.Fprintf(buf, "%d:%s", len(val), val)
	case int:
		fmt.Fprintf(buf, "i%de", val)
	case []interface{}:
		buf.WriteByte('l')
		for _, item := range val {
			if err := marshalTo(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte('e')
	case map[string]interface{}:
		buf.WriteByte('d')
		// Keys must be sorted
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			// Marshal key (must be a string)
			fmt.Fprintf(buf, "%d:%s", len(k), k)
			// Marshal value
			if err := marshalTo(buf, val[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('e')
	default:
		return fmt.Errorf("bencode: unsupported type for marshaling: %T", v)
	}
	return nil
}
