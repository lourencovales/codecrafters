package bencode

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"unicode"
)

type decoder struct {
	data []byte
	pos  int
}

func Unmarshal(data []byte) (interface{}, error) {

	if len(data) == 0 {
		return nil, fmt.Errorf("bencode: cannot unmarshal empty data")
	}
	d := &decoder{data: data, pos: 0}
	return d.decode()
}

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

func (d *decoder) decodeString() (string, error) {
	colonIndex := bytes.IndexByte(d.data[d.pos:], ':')
	if colonIndex == -1 {
		return "", fmt.Errorf("bencode: invalid string format, missing colon")
	}
	colonIndex += d.pos

	lengthStr := string(d.data[d.pos:colonIndex])
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", fmt.Errorf("bencode: invalid string length '%s'", lengthStr)
	}

	start := colonIndex + 1
	end := start + length
	if end > len(d.data) {
		return "", fmt.Errorf("bencode: string length exceeds data boundary")
	}

	d.pos = end
	return string(d.data[start:end]), nil
}

func (d *decoder) decodeInt() (int, error) {
	d.pos++ // Skip 'i'
	endIndex := bytes.IndexByte(d.data[d.pos:], 'e')
	if endIndex == -1 {
		return 0, fmt.Errorf("bencode: invalid integer format, missing 'e'")
	}
	endIndex += d.pos

	intStr := string(d.data[d.pos:endIndex])
	val, err := strconv.Atoi(intStr)
	if err != nil {
		return 0, fmt.Errorf("bencode: invalid integer value '%s'", intStr)
	}

	d.pos = endIndex + 1
	return val, nil
}

func (d *decoder) decodeList() ([]interface{}, error) {
	d.pos++ // Skip 'l'
	var list []interface{}
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

func (d *decoder) decodeDict() (map[string]interface{}, error) {
	d.pos++ // Skip 'd'
	dict := make(map[string]interface{})
	for d.pos < len(d.data) && d.data[d.pos] != 'e' {
		key, err := d.decodeString()
		if err != nil {
			return nil, err
		}
		val, err := d.decode()
		if err != nil {
			return nil, err
		}
		dict[key] = val
	}

	if d.pos >= len(d.data) || d.data[d.pos] != 'e' {
		return nil, fmt.Errorf("bencode: invalid dictionary format, missing 'e'")
	}

	d.pos++ // Skip 'e'
	return dict, nil
}

func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := marshalTo(&buf, v)
	return buf.Bytes(), err
}

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
