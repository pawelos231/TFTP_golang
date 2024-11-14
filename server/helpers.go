package server

import (
	"bytes"
	"errors"
	"fmt"
)

func decodeNetAscii(data []byte) (string, error) {
	buf := new(bytes.Buffer)
	i := 0
	len := len(data)
	for i < len {
		if data[i] == '\r' {
			if i+1 >= len {
				return "", errors.New("Invalid netascii")
			}
			nextByte := data[i+1]
			if nextByte == '\n' {
				// CR LF sequence represents newline
				buf.WriteByte('\n')
				i += 2
			} else if nextByte == 0 {
				// CR NUL sequence represents carriage return
				buf.WriteByte('\r')
				i += 2
			} else {
				return "", fmt.Errorf("Invalid NetASCII sequence: 0x%X after CR", nextByte)
			}
		} else {
			buf.WriteByte(data[i])
			i++
		}
	}
	return buf.String(), nil

}

func encodeNetAscii(message string) ([]byte, error) {
	buf := new(bytes.Buffer)
	for i := 0; i < len(message); i++ {
		if message[i] == '\n' {
			buf.WriteByte('\r')
			buf.WriteByte('\n')
		} else if message[i] == '\r' {
			buf.WriteByte('\r')
			buf.WriteByte(0)
		} else {
			buf.WriteByte(message[i])
		}
	}
	return buf.Bytes(), nil
}
