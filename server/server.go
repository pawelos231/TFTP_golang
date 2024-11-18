package server

import (
	"errors"
	"net"
	"time"
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
	// continue implementation

	return nil
}
