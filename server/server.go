package server

import (
	"errors"
	"log"
	"net"
	"time"
)

type FileType int

const (
	Binary = iota
	Netascii
)

type Server struct {
	timeout time.Duration
	Retries int
	Payload []byte
}

func (s *Server) ListenAndServe(addr string) error {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return errors.New("Error listening on address")
	}
	defer conn.Close()
	s.Serve(conn)

	return nil
}

func (s *Server) Serve(conn net.PacketConn) error {
	if conn == nil {
		return errors.New("Invalid connection")
	}

	if s.Payload == nil {
		return errors.New("Invalid payload")
	}

	if s.Retries < 0 {
		s.Retries = 10
	}

	if s.timeout == 0 {
		s.timeout = time.Second * 10
	}
	var readReq ReadRequest
	var writeReq WriteRequest

	for {
		buf := make([]byte, 1024)
		_, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return errors.New("Error reading from connection")
		}

		err = readReq.UnmarshalBinary(s.Payload)
		if err == nil {
			go s.handle(readReq, addr, Binary)
		}
		err = readReq.UmarshalNetascii(s.Payload)
		if err == nil {
			go s.handle(readReq, addr, Netascii)
		}

		err = writeReq.UnmarshalBinary(s.Payload)
		if err == nil {
			go s.handle(writeReq, addr, Binary)
		}

		err = writeReq.UmarshalNetascii(s.Payload)
		if err == nil {
			go s.handle(writeReq, addr, Netascii)
		}
	}
}

func (s *Server) handle(rrq Request, addr net.Addr, typ FileType) {
	switch rrq.(type) {
	case ReadRequest:
		s.handleReadRequest(rrq.(ReadRequest), addr, typ)
	case WriteRequest:
		s.handleWriteRequest(rrq.(WriteRequest), addr, typ)
	}
}

func (s *Server) handleReadRequest(rrq ReadRequest, addr net.Addr, typ FileType) {
	log.Printf("[%s] requested file: %s", addr, rrq.FileName)
	//we create a new connection to the client, beacuse by creating a new connection we can send a file to the correct client
	//and we do not need to worry about synchronization issues with the "connection" from net.ListenPacket in the Serve method
	conn, err := net.Dial("udp", addr.String())
	if err != nil {
		log.Printf("Error connecting to client: %v", err)
		return
	}

	defer func() { _ = conn.Close() }()

	switch typ {
	case Binary:
		// handle binary
	case Netascii:
		// handle netascii
	}
}
func (s *Server) handleWriteRequest(wrq WriteRequest, addr net.Addr, typ FileType) {

	log.Printf("[%s] requested file: %s", addr, wrq.FileName)
	//we create a new connection to the client, beacuse by creating a new connection we can send a file to the correct client
	//and we do not need to worry about synchronization issues with the "connection" from net.ListenPacket in the Serve method
	conn, err := net.Dial("udp", addr.String())
	if err != nil {
		log.Printf("Error connecting to client: %v", err)
		return
	}
	defer func() { _ = conn.Close() }()

	switch typ {
	case Binary:
		// handle binary
	case Netascii:
		// handle netascii
	}
}
