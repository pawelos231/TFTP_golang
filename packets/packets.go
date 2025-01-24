package packets

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

const (
	DatagramSize = 516
	BlockSize    = DatagramSize - 4 // datagram size minus the opcode and block number
	NETASCII     = "netascii"
	OCTET        = "octet"
)

type OpCode uint16
type Compress bool 

// types of packets (opcodes) that can be sent
const (
	PRQ   OpCode = iota + 1 // Read request
	WRQ                     // Write request
	DATA                    // Data
	ACK                     // Acknowledgement
	ERROR                   // Error
)

type ErrCode uint16

const (
	ErrUnknown         ErrCode = iota // Not defined, see error message (if any).
	ErrNotFound                       // File not found.
	ErrAccessViolation                // Access violation.
	ErrDiskFull                       // Disk full or allocation exceeded.
	ErrIllegalOp                      // Illegal TFTP operation.
	ErrUnknownID                      // Unknown transfer ID.
	ErrFileExists                     // File already exists.
	ErrNoUser                         // No such user.
)

type Request interface {
    RequestType() string
    MarshalBinary() ([]byte, error)
    MarshalNetascii() ([]byte, error)
}

// READ REQUEST PACKET
type ReadRequest struct {
	FileName string // name of the file to read
	Mode     string // "netascii", "octet"
	Compress bool   // compress the file (that is a twist in the protocol)
}

func (r ReadRequest) RequestType() string {
	return "read"
}

func (r *ReadRequest) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	var code OpCode
	var compress Compress


	//read opcode
	err := binary.Read(buf, binary.BigEndian, &code)
	if err != nil {
		return errors.New("Invalid opcode")
	}


	if code != PRQ {
		return errors.New("Invalid PRQ ")
	}

	//read compress, see how the packet is structured in the README
	err = binary.Read(buf, binary.BigEndian, &compress)
	if err != nil {
		return errors.New("Invalid compress")
	}

	r.FileName, err = buf.ReadString(0)
	if err != nil {
		return errors.New("Invalid filename")
	}

	r.FileName = strings.TrimRight(r.FileName, "\x00")
	if r.FileName == "" {
		return errors.New("Invalid filename")
	}



	r.Mode, err = buf.ReadString(0)
	if err != nil {
		return errors.New("Invalid PRQ")
	}

	r.Mode = strings.TrimRight(r.Mode, "\x00")
	if r.Mode != NETASCII && r.Mode != OCTET {
		return errors.New("Invalid mode")
	}


	return nil
}

func (r ReadRequest) MarshalBinary() ([]byte, error) {
	mode := "octet"
	compress := false
	if r.Mode != "" {
		mode = r.Mode
	}

	if r.Compress {
		compress = r.Compress
	}

	cap := 2 + 1 + 2 + len(r.FileName) + 1 + len(mode) + 1
	buf := new(bytes.Buffer)
	buf.Grow(cap)

	var code OpCode = PRQ
	err := binary.Write(buf, binary.BigEndian, code)
	if err != nil {
		return nil, err
	}


	var compressByte byte
	if compress {
		compressByte = 1
	} else {
		compressByte = 0
	}
	err = binary.Write(buf, binary.BigEndian, compressByte)

	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte(r.FileName))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte{0})
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte(r.Mode))
	if err != nil {
		return nil, err
	}


	return buf.Bytes(), nil

}
func (r *ReadRequest) UnmarshalNetascii(data []byte) error {
	buf := bytes.NewBuffer(data)
	var code OpCode
	var compress Compress

	//read opcode
	err := binary.Read(buf, binary.BigEndian, &code)
	if err != nil {
		return errors.New("Invalid opcode")
	}

	//read compress
	err = binary.Read(buf, binary.BigEndian, &compress)
	if err != nil {
		return errors.New("Invalid compress")
	}

	if code != PRQ {
		return errors.New("Invalid PRQ")
	}

	netAsciiFileName, err := buf.ReadBytes(0)
	if err != nil {
		return errors.New("Invalid filename")
	}

	// Remove the null terminator
	if len(netAsciiFileName) > 0 {
		netAsciiFileName = netAsciiFileName[:len(netAsciiFileName)-1]
	}

	r.FileName, err = decodeNetAscii(netAsciiFileName)
	if err != nil {
		return errors.New("Invalid filename")
	}

	netAsciiMode, err := buf.ReadBytes(0)
	if err != nil {
		return errors.New("Invalid mode")
	}

	// Remove the null terminator
	if len(netAsciiMode) > 0 {
		netAsciiMode = netAsciiMode[:len(netAsciiMode)-1]
	}

	r.Mode, err = decodeNetAscii(netAsciiMode)
	if err != nil {
		return errors.New("Invalid mode")
	}

	return nil
}

func (r ReadRequest) MarshalNetascii() ([]byte, error) {
	buf := new(bytes.Buffer)
	cap := 2 + 2 + len(r.FileName) + 1 + len(r.Mode) + 1
	buf.Grow(cap)
	const code = PRQ

	err := binary.Write(buf, binary.BigEndian, code)
	if err != nil {
		return nil, err
	}

	netAsciiFileName, err := encodeNetAscii(r.FileName)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, netAsciiFileName)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte{0})
	if err != nil {
		return nil, err
	}

	netAsciiMode, err := encodeNetAscii(r.Mode)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, netAsciiMode)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte{0})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// WRITE REQUEST PACKET
type WriteRequest struct {
	FileName string // name of the file to write
	Mode     string // "netascii", "octet"
	Compress bool   // compress the file (that is a twist in the protocol)
}

func (w WriteRequest) RequestType() string {
	return "write"
}

func (w *WriteRequest) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	var code OpCode
	var compress Compress

	//read opcode
	err := binary.Read(buf, binary.BigEndian, &code)
	if err != nil {
		return errors.New("Invalid opcode")
	}

	//read compress
	err = binary.Read(buf, binary.BigEndian, &compress)
	if err != nil {
		return errors.New("Invalid compress")
	}

	if code != WRQ {
		return errors.New("Invalid PRQ or WRQ")
	}

	w.FileName, err = buf.ReadString(0)
	if err != nil {
		return errors.New("Invalid filename")
	}

	w.FileName = strings.TrimRight(w.FileName, "\x00")
	if w.FileName == "" {
		return errors.New("Invalid filename")
	}

	w.Mode, err = buf.ReadString(0)
	if err != nil {
		return errors.New("Invalid PRQ")
	}

	w.Mode = strings.TrimRight(w.Mode, "\x00")
	if w.Mode != OCTET {
		return errors.New("Invalid mode")
	}

	return nil
}

func (w WriteRequest) MarshalBinary() ([]byte, error) {
	mode := "octet"
	if w.Mode != "" {
		mode = w.Mode
	}

	cap := 2 + 2 + len(w.FileName) + 1 + len(mode) + 1
	buf := new(bytes.Buffer)
	buf.Grow(cap)

	var code OpCode = WRQ
	err := binary.Write(buf, binary.BigEndian, code)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte(w.FileName))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte{0})
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte(w.Mode))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte{0})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (w *WriteRequest) UnmarshalNetascii(data []byte) error {
	buf := bytes.NewBuffer(data)
	var code OpCode
	var compress Compress

	//read opcode
	err := binary.Read(buf, binary.BigEndian, &code)
	if err != nil {
		return errors.New("Invalid opcode")
	}

	//read compress
	err = binary.Read(buf, binary.BigEndian, &compress)
	if err != nil {
		return errors.New("Invalid compress")
	}

	if code != WRQ {
		return errors.New("Invalid WRQ")
	}

	netAsciiFileName, err := buf.ReadBytes(0)
	if err != nil {
		return errors.New("Invalid filename")
	}
	// Remove the null terminator
	if len(netAsciiFileName) > 0 {
		netAsciiFileName = netAsciiFileName[:len(netAsciiFileName)-1]
	}

	w.FileName, err = decodeNetAscii(netAsciiFileName)
	if err != nil {
		return errors.New("Invalid filename")
	}

	netAsciiMode, err := buf.ReadBytes(0)
	if err != nil {
		return errors.New("Invalid mode")
	}

	// Remove the null terminator
	if len(netAsciiMode) > 0 {
		netAsciiMode = netAsciiMode[:len(netAsciiMode)-1]
	}

	w.Mode, err = decodeNetAscii(netAsciiMode)
	if err != nil {
		return errors.New("Invalid mode")
	}

	if w.Mode != NETASCII {
		return errors.New("Invalid mode")
	}

	return nil
}

func (w WriteRequest) MarshalNetascii() ([]byte, error) {
	buf := new(bytes.Buffer)
	cap := 2 + 2 + len(w.FileName) + 1 + len(w.Mode) + 1
	buf.Grow(cap)
	const code = WRQ

	err := binary.Write(buf, binary.BigEndian, code)
	if err != nil {
		return nil, err
	}

	netAsciiFileName, err := encodeNetAscii(w.FileName)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, netAsciiFileName)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte{0})
	if err != nil {
		return nil, err
	}

	netAsciiMode, err := encodeNetAscii(w.Mode)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, netAsciiMode)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte{0})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// DATA PACKET
type Data struct {
	BlockNumber uint16    // block number of the data packet
	Payload     io.Reader // payload of the data packet
}

func (d *Data) UnmarshalBinary(data []byte) error {
	len := len(data)
	if len < 4 || len > DatagramSize {
		return errors.New("Invalid data packet")
	}

	buf := bytes.NewBuffer(data)
	var code OpCode

	err := binary.Read(bytes.NewReader(buf.Next(2)), binary.BigEndian, &code)
	if err != nil {
		return errors.New("Invalid OpCode")
	}

	if code != DATA {
		return errors.New("Invalid OpCode")
	}

	err = binary.Read(bytes.NewReader(buf.Next(2)), binary.BigEndian, &d.BlockNumber)
	if err != nil {
		return errors.New("Invalid block number")
	}

	d.Payload = buf

	return nil
}
func (d Data) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	cap := 2 + 2 + DatagramSize
	buf.Grow(cap)

	var code OpCode = DATA
	err := binary.Write(buf, binary.BigEndian, code)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, d.BlockNumber)
	if err != nil {
		return nil, err
	}

	_, err = io.CopyN(buf, d.Payload, BlockSize)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buf.Bytes(), nil

}

// ACK PACKET
type Ack struct {
	BlockNumber uint16 // block number of the data packet
}

func (a *Ack) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	var code OpCode

	err := binary.Read(bytes.NewReader(buf.Next(2)), binary.BigEndian, &code)
	if err != nil {
		return errors.New("Invalid OpCode")
	}

	if code != ACK {
		return errors.New("Invalid OpCode")
	}

	err = binary.Read(bytes.NewReader(buf.Next(2)), binary.BigEndian, &a.BlockNumber)
	if err != nil {
		return errors.New("Invalid block number")
	}

	return nil
}

func (a Ack) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	cap := 2 + 2
	buf.Grow(cap)

	code := ACK
	err := binary.Write(buf, binary.BigEndian, code)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, a.BlockNumber)
	if err != nil {
		return nil, err
	}


	return buf.Bytes(), nil
}

// ERROR PACKET
type Error struct {
	ErrCode ErrCode // error code
	Message string  // error message
}

func (e *Error) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	var code OpCode
	err := binary.Read(buf, binary.BigEndian, &code)
	if err != nil {
		return errors.New("Invalid opcode")
	}

	if code != ERROR {
		return errors.New("Invalid error packet")
	}

	err = binary.Read(buf, binary.BigEndian, &e.ErrCode)
	if err != nil {
		return errors.New("Invalid error code")
	}

	e.Message, err = buf.ReadString(0)
	e.Message = strings.TrimRight(e.Message, "\x00") // remove the 0-byte
	if err != nil {
		return errors.New("Invalid error message")
	}

	return nil
}

func (e Error) MarshalBinary() ([]byte, error) {
	cap := 2 + 2 + len(e.Message) + 1
	buf := new(bytes.Buffer)
	buf.Grow(cap)

	code := ERROR
	err := binary.Write(buf, binary.BigEndian, code)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, e.ErrCode)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, []byte(e.Message))
	if err != nil {
		return nil, err
	}
	err = buf.WriteByte(0)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
func (e *Error) UmarshalNetascii(data []byte) error {
	buf := bytes.NewBuffer(data)
	var code OpCode

	err := binary.Read(buf, binary.BigEndian, &code)
	if err != nil {
		return errors.New("Invalid opcode")
	}

	if code != ERROR {
		return errors.New("Invalid error packet")
	}
	err = binary.Read(buf, binary.BigEndian, &e.ErrCode)
	if err != nil {
		return errors.New("Invalid error code")
	}

	netasciiData, err := buf.ReadBytes(0)
	if err != nil {
		return errors.New("Invalid error message")
	}

	// Remove the null terminator
	if len(netasciiData) > 0 {
		netasciiData = netasciiData[:len(netasciiData)-1]
	}

	e.Message, err = decodeNetAscii(netasciiData)
	if err != nil {
		return errors.New("Invalid error message")
	}

	return nil
}

func (e Error) MarshalNetascii() ([]byte, error) {
	buf := new(bytes.Buffer)
	cap := 2 + 2 + len(e.Message) + 1
	buf.Grow(cap)
	const code = ERROR

	err := binary.Write(buf, binary.BigEndian, code)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, e.ErrCode)
	if err != nil {
		return nil, err
	}

	netasciiData, err := encodeNetAscii(e.Message)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, netasciiData)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
