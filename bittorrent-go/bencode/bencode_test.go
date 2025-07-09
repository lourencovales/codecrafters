package bencode

import (
	"reflect"
	"testing"
)

func TestUnmarshalString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple string", "4:spam", "spam"},
		{"empty string", "0:", ""},
		{"longer string", "11:hello world", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Unmarshal([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestUnmarshalInteger(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"positive integer", "i42e", 42},
		{"negative integer", "i-42e", -42},
		{"zero", "i0e", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Unmarshal([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %d, got %v", tt.expected, result)
			}
		})
	}
}

func TestUnmarshalList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []interface{}
	}{
		{"empty list", "le", []interface{}{}},
		{"string list", "l4:spam4:eggse", []interface{}{"spam", "eggs"}},
		{"mixed list", "l4:spami42ee", []interface{}{"spam", 42}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Unmarshal([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resultSlice, ok := result.([]interface{})
			if !ok {
				t.Fatalf("expected []interface{}, got %T", result)
			}
			if len(resultSlice) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(resultSlice))
			}
			for i, v := range resultSlice {
				if !reflect.DeepEqual(v, tt.expected[i]) {
					t.Errorf("at index %d: expected %v, got %v", i, tt.expected[i], v)
				}
			}
		})
	}
}

func TestUnmarshalDict(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{"empty dict", "de", map[string]interface{}{}},
		{"simple dict", "d3:cow3:moo4:spam4:eggse", map[string]interface{}{"cow": "moo", "spam": "eggs"}},
		{"mixed dict", "d4:spami42e3:cow3:mooe", map[string]interface{}{"spam": 42, "cow": "moo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Unmarshal([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestUnmarshalErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty input", ""},
		{"invalid string format", "4spam"},
		{"invalid integer format", "i42"},
		{"invalid list format", "l4:spam"},
		{"invalid dict format", "d3:cow"},
		{"invalid character", "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Unmarshal([]byte(tt.input))
			if err == nil {
				t.Errorf("expected error for input %q", tt.input)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "spam", "4:spam"},
		{"integer", 42, "i42e"},
		{"list", []interface{}{"spam", 42}, "l4:spami42ee"},
		{"dict", map[string]interface{}{"cow": "moo", "spam": "eggs"}, "d3:cow3:moo4:spam4:eggse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestMarshalUnsupportedType(t *testing.T) {
	_, err := Marshal(3.14)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []interface{}{
		"hello",
		42,
		[]interface{}{"spam", "eggs", 123},
		map[string]interface{}{"key": "value", "number": 42},
	}

	for _, tt := range tests {
		t.Run("roundtrip", func(t *testing.T) {
			encoded, err := Marshal(tt)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}
			decoded, err := Unmarshal(encoded)
			if err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			if !reflect.DeepEqual(tt, decoded) {
				t.Errorf("roundtrip failed: expected %v, got %v", tt, decoded)
			}
		})
	}
}