package server

import "testing"

func TestDecodeNetAscii(t *testing.T) {
	data := []byte("Hello\r\nWorld\r\n")
	expected := "Hello\nWorld\n"
	actual, err := decodeNetAscii(data)
	if err != nil {
		t.Errorf("Error decoding NetASCII: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

func TestDecodeNetAsciiInvalid(t *testing.T) {
	data := []byte("Hello\rWorld\r\n")
	_, err := decodeNetAscii(data)
	if err == nil {
		t.Errorf("Expected error decoding NetASCII")
	}
}

func TestDecodeNetAsciiInvalidSequence(t *testing.T) {
	data := []byte("Hello\r\x0DWorld\r\n")
	_, err := decodeNetAscii(data)
	if err == nil {
		t.Errorf("Expected error decoding NetASCII")
	}
}

func TestEncocdeAscii(t *testing.T) {
	message := "Hello\nWorld\n"
	expected := []byte("Hello\r\nWorld\r\n")
	actual, err := encodeNetAscii(message)
	if err != nil {
		t.Errorf("Error encoding NetASCII: %v", err)
	}

	if len(actual) != len(expected) {
		t.Errorf("Expected %d bytes, got %d", len(expected), len(actual))
	}

	if string(actual) != string(expected) {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}
