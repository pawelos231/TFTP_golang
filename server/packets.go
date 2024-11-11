package server

import "io"

const (
	DatagramSize = 516
	BlockSize    = DatagramSize - 4 // datagram size minus the opcode and block number
)

type OpCode uint16

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

type ReadRequest struct {
	FileName string // name of the file to read
	Mode     string // "netascii", "octet", "mail"
}

func (r *ReadRequest) UnmarshalBinary(data []byte) error { return nil }
func (r ReadRequest) MarshalBinary() ([]byte, error)     { return nil, nil }

type WriteRequest struct {
	FileName string // name of the file to write
	Mode     string // "netascii", "octet", "mail"
}

func (w *WriteRequest) UnmarshalBinary(data []byte) error { return nil }
func (w WriteRequest) MarshalBinary() ([]byte, error)     { return nil, nil }

type Data struct {
	BlockNumber uint16    // block number of the data packet
	Payload     io.Reader // payload of the data packet
}

func (d *Data) UnmarshalBinary(data []byte) error { return nil }
func (d Data) MarshalBinary() ([]byte, error)     { return nil, nil }

type Ack struct {
	BlockNumber uint16 // block number of the data packet
}

func (a *Ack) UnmarshalBinary(data []byte) error { return nil }
func (a Ack) MarshalBinary() ([]byte, error)     { return nil, nil }

type Error struct {
	ErrCode ErrCode // error code
	Message string  // error message
}
