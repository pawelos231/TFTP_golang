package server

import (
	"TFTP/packets"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

type FileType int

const (
	Binary = iota
	Netascii
)

type Server struct {
	Timeout time.Duration
	Retries int
}

func (s *Server) ListenAndServe(addr string) error {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return errors.New("Error listening on address")
	}
	defer func() { _ = conn.Close() }()
	log.Printf("Listening on %s ...\n", conn.LocalAddr())

	return s.Serve(conn)
}

func (s *Server) Serve(conn net.PacketConn) error {
	if conn == nil {
		return errors.New("Invalid connection")
	}

	if s.Retries < 0 {
		s.Retries = 10
	}

	if s.Timeout == 0 {
		s.Timeout = time.Second * 10
	}
	var readReq packets.ReadRequest
	var writeReq packets.WriteRequest

	for {
		buf := make([]byte, 1024)
		_, client_addr, err := conn.ReadFrom(buf)
		if err != nil {
			return errors.New("Error reading from connection")
		}
		fmt.Printf("Received request from: %v", string(buf))

		err = readReq.UnmarshalBinary(buf)
		if err == nil {
			go s.handle(readReq, client_addr)
			continue
		}
		err = readReq.UnmarshalNetascii(buf)
		if err == nil {
			go s.handle(readReq, client_addr)
			continue
		}

		err = writeReq.UnmarshalBinary(buf)
		if err == nil {
			go s.handle(writeReq, client_addr)
			continue
		}

		err = writeReq.UnmarshalNetascii(buf)
		if err == nil {
			go s.handle(writeReq, client_addr)
			continue
		}

		// log.Printf("Invalid request: %v, buffer: %s", err, buf)
		// return err //returning error beacuse we do not want to continue the server if we have an invalid request
	}
}

func (s *Server) handle(rrq packets.Request, client_addr net.Addr) {
	switch rrq.(type) {
	case packets.ReadRequest:
		s.handleReadRequest(rrq.(packets.ReadRequest), client_addr)
	case packets.WriteRequest:
		s.handleWriteRequest(rrq.(packets.WriteRequest), client_addr)
	}
}

func (s *Server) handleReadRequest(rrq packets.ReadRequest, client_addr net.Addr) {
	log.Printf("[%s] requested file: %s", client_addr, rrq.FileName)
	//we create a new connection to the client, beacuse by creating a new connection we can send a file to the correct client
	//and we do not need to worry about synchronization issues with the "connection" from net.ListenPacket in the Serve method
	conn, err := net.Dial("udp", client_addr.String())
	if err != nil {
		log.Printf("Error connecting to client: %v", err)
		return
	}

	defer func() { _ = conn.Close() }()

	if rrq.Compress {
		//TODO: implement file compression
	}

	payload, err := os.ReadFile(rrq.FileName)
	if err != nil {
		fmt.Println("Error reading payload file")
		return
	}

	var (
		ackPacket   packets.Ack
		errorPacket packets.Error
		dataPacket  = packets.Data{Payload: bytes.NewReader(payload)}
		buf         = make([]byte, packets.DatagramSize)
	)

NEXT:
	//keep sending data packets until we reach the end of the file
	//so until n == DatagramSize beacuse when n gets smaller that means we reached the end of the file
	for n := packets.DatagramSize; n == packets.DatagramSize; {
		data, err := dataPacket.MarshalBinary()
		if err != nil {
			log.Printf("Error marshaling data packet: %v", err)
			return
		}

	RETRY:
		for i := 0; i < s.Retries; i++ {
			n, err = conn.Write(data)
			if err != nil {
				log.Printf("Error sending data packet: %v", err)
				return
			}

			_ = conn.SetReadDeadline(time.Now().Add(s.Timeout))

			//we read the ACK packet from the client
			_, err = conn.Read(buf)
			if err != nil {
				if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
					log.Printf("Timeout reading ack packet: %v", err)
					continue RETRY
				}
				log.Printf("Error unmarshaling ack packet: %v", err)
				return
			}

			ackErr := ackPacket.UnmarshalBinary(buf)
			errorErr := errorPacket.UnmarshalBinary(buf)

			switch {
			case ackErr == nil:
				if uint16(ackPacket.BlockNumber) == dataPacket.BlockNumber {
					continue NEXT
				} else {
					log.Printf("Unexpected ACK block number: got %d, expected %d", ackPacket.BlockNumber, dataPacket.BlockNumber)
					break
				}
			case errorErr == nil:
				log.Printf("Error packet received: %v", errorPacket)
				return
			default:
				log.Printf("Invalid packet received: %v", buf)
			}

			log.Printf("Max retries reached for: %s", client_addr)
			return
		}

	}
	log.Printf("[%s] file sent", client_addr)

}

func (s *Server) handleWriteRequest(wrq packets.WriteRequest, client_addr net.Addr) {

	log.Printf("[%s] adding file: %s", client_addr, wrq.FileName)
	// we create a new connection to the client, beacuse by creating a new connection we can send a file to the correct client
	// and we do not need to worry about synchronization issues with the "connection" from net.ListenPacket in the Serve method
	// Bind to a local ephemeral port
	conn, err := net.Dial("udp", client_addr.String())
	if err != nil {
		log.Printf("Error connecting to client: %v", err)
		return
	}

	defer func() { _ = conn.Close() }()

	// Send initial packet to client
	// This is to let the client know the new port to connect to
	_, err = conn.Write([]byte{0})
	if err != nil {
		log.Printf("Error sending initial packet: %v", err)
		return
	}

	log.Printf("Local connection created on %s", conn.LocalAddr())

	defer func() { _ = conn.Close() }()

	if wrq.Compress {
		// TODO: implement file compression
	}

	var (
		ackPacket   packets.Ack
		errorPacket packets.Error
		dataPacket  packets.Data
		buf         = make([]byte, packets.DatagramSize)
	)

	//create a file to write to
	//in case of error we close and destroy the file
	file, err := os.Create("received" + wrq.FileName)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return
	}

	//Ensure file is closed and deleted on failure
	defer func() {
		if file != nil {
			file.Close()
			if err != nil {
				if removeErr := os.Remove(wrq.FileName); removeErr != nil {
					log.Printf("Failed to delete incomplete file '%s': %v", wrq.FileName, removeErr)
				} else {
					log.Printf("Incomplete file '%s' deleted due to errors.", wrq.FileName)
				}
			}
		}
	}()

GET_NEXT:
	for {
		err = conn.SetReadDeadline(time.Now().Add(s.Timeout))
		if err != nil {
			log.Printf("Error setting read deadline: %v", err)
			return
		}

		n, err := conn.Read(buf)
		if err != nil {
			if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
				log.Printf("Timeout reading data packet: %v", err)
				//might need to break out after a certain number of retries
				continue GET_NEXT
			}
			log.Printf("Error reading data packet: %v", err)
			return
		}

		dataErr := dataPacket.UnmarshalBinary(buf)
		errorErr := errorPacket.UnmarshalBinary(buf)
		switch {
		case dataErr == nil:
			{
				fmt.Printf("Data packet received: %v\n", dataPacket)

				dataSize := n - 4

				_, err = file.Write(buf[4 : 4+dataSize])
				if err != nil {
					log.Printf("Error writing to file: %v", err)
					return
				}

				ackPacket.BlockNumber = dataPacket.BlockNumber

			}

		case errorErr == nil:
			{
				err = file.Close()
				if err != nil {
					log.Printf("Error closing file: %v", err)
					return
				}
				log.Printf("Error packet received: %v", errorPacket)
				return
			}
		}

		marshaledAck, err := ackPacket.MarshalBinary()
		if err != nil {
			log.Printf("Error marshaling ack packet: %v", err)
			return
		}

		_, err = conn.Write(marshaledAck)
		if err != nil {
			log.Printf("Error sending ack packet: %v", err)
			return
		}

	}

}
