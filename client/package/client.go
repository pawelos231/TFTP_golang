package client

import (
	"TFTP/packets"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	opcodeRRQ   = 1
	opcodeWRQ   = 2
	opcodeDATA  = 3
	opcodeACK   = 4
	opcodeERROR = 5
)

func SendRequest(req packets.Request, serverIP *string) (*net.UDPConn, *net.UDPAddr, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", *serverIP)
	if err != nil {
		fmt.Println("Invalid server address:", err)
		return nil, nil, err
	}

	// Set up local UDP connection, any available local address
	localConn, err := net.ListenUDP("udp", nil) // nil means any available local address
	if err != nil {
		fmt.Println("Failed to set up local UDP connection:", err)
		return nil, nil, err
	}
	fmt.Printf("Local UDP connection set up on %s\n", localConn.LocalAddr())

	reqData, err := req.MarshalBinary()
	if err != nil {
		fmt.Println("Error while marshaling REQ:", err)
		return nil, nil, err
	}
	fmt.Printf("Sending %s request to %s\n", req.String(), serverAddr)

	// Send REQ to server
	_, err = localConn.WriteTo(reqData, serverAddr)
	if err != nil {
		fmt.Println("Error while sending REQ:", err)
		return nil, nil, err
	}
	return localConn, serverAddr, nil
}

type Handler struct {
	Conn     *net.UDPConn
	Addr     *net.UDPAddr
	Deadline time.Duration
}

func NewHandler(conn *net.UDPConn, addr *net.UDPAddr, deadline time.Duration) *Handler {
	return &Handler{
		Conn:     conn,
		Addr:     addr,
		Deadline: deadline,
	}
}

func (h *Handler) HandleReadRequest(filename *string, transferSucessful chan bool) error {
	// Open file for writing the received data
	outputFileName := "received_" + *filename
	outputFile, err := os.Create(strings.ReplaceAll(outputFileName, "/", "_"))
	if err != nil {
		return err
	}
	// Ensure file is closed and deleted on failure
	defer func() {
		if outputFile != nil {
			outputFile.Close()
			if err != nil {
				if removeErr := os.Remove(outputFileName); removeErr != nil {
					log.Printf("Failed to delete incomplete file '%s': %v", outputFileName, removeErr)
				} else {
					log.Printf("Incomplete file '%s' deleted due to errors.", outputFileName)
				}
			}
		}
	}()
	fmt.Printf("Output file created: %s\n", outputFile.Name())

	// Variables to track the server's ephemeral address
	var serverDataAddr *net.UDPAddr

	for {
		buffer := make([]byte, 516)
		h.Conn.SetReadDeadline(time.Now().Add(h.Deadline))

		n, addr, err := h.Conn.ReadFromUDP(buffer)
		if err != nil {
			return err
		}

		// Update serverDataAddr if it's the first data packet
		if serverDataAddr == nil {
			serverDataAddr = addr
			fmt.Printf("Server data address set to %s\n", serverDataAddr)
		}

		if !addr.IP.Equal(serverDataAddr.IP) || addr.Port != serverDataAddr.Port {
			fmt.Printf("Received packet from unknown address %s\n", addr)
			continue // Ignore packets from unknown sources
		}

		opcode := buffer[1]
		switch opcode {
		case opcodeDATA:
			dataPck := packets.Data{}
			err = dataPck.UnmarshalBinary(buffer[:n])
			if err != nil {
				fmt.Println("Error unmarshaling DATA packet:", err)
				break
			}

			// Write data to file
			// TFTP DATA packet: 2 bytes opcode + 2 bytes block number + data
			_, err = outputFile.Write(buffer[4:n])
			if err != nil {
				return err
			}

			// Send ACK
			ack := packets.Ack{BlockNumber: dataPck.BlockNumber}
			ackData, err := ack.MarshalBinary()
			if err != nil {
				return fmt.Errorf("Error while marshaling ACK packet: %v", err)
			}

			b, err := h.Conn.WriteTo(ackData, serverDataAddr)
			fmt.Printf("Sent ACK for block %d, sent %d\n", dataPck.BlockNumber, b)
			if err != nil {
				return fmt.Errorf("Error while sending ACK packet: %v", err)
			}

			if n < 516 {
				transferSucessful <- true
				break
			}

		case opcodeERROR:
			errorPck := packets.Error{}
			err = errorPck.UnmarshalBinary(buffer[:n])
			if err != nil {
				return fmt.Errorf("Error unmarshaling ERROR packet: %v", err)
			} else {
				return fmt.Errorf("Received ERROR packet: %s", errorPck.Message)
			}

		default:
			fmt.Printf("Unknown opcode %d received\n", opcode)
		}

		if opcode == opcodeERROR || err != nil {
			break
		}

	}

	// Close file and return
	if err != nil {
		log.Printf("File '%s' received successfully.", outputFileName)
		err = outputFile.Close()
		if err != nil {
			log.Printf("Error closing file '%s': %v", outputFileName, err)
		}
		outputFile = nil
	} else {
		log.Printf("File transfer failed. Cleaning up '%s'.", outputFileName)
	}

	return nil
}

func (h *Handler) HandleWriteRequest(filename *string, transferSucessful chan bool) error {
	log.Printf("Handling write request for file: %s", *filename)
	//read file
	payload, err := os.ReadFile(*filename)
	if err != nil {
		fmt.Println("Error reading payload file")
		return err
	}

	var (
		ackPacket   packets.Ack
		errorPacket packets.Error
		dataPacket  = packets.Data{Payload: bytes.NewReader(payload)}
		buf         = make([]byte, packets.DatagramSize)
	)

	// we read the initial packet from the server
	// we do it only to get the server address
	_, addr, err := h.Conn.ReadFromUDP(buf)
	if err != nil {
		return err
	}

NEXT:
	for n := packets.DatagramSize; n == packets.DatagramSize; {
		data, err := dataPacket.MarshalBinary()
		if err != nil {
			log.Printf("Error marshaling data packet: %v", err)
			return err
		}

	RETRY:
		for i := 0; i < 10; i++ {
			n, err = h.Conn.WriteTo(data, addr)
			if err != nil {
				log.Printf("Error sending data packet: %v", err)
				return err
			}

			// dataPacket.BlockNumber++
			h.Conn.SetReadDeadline(time.Now().Add(h.Deadline / 10))

			// we read ACK packet from server
			_, addr, err = h.Conn.ReadFromUDP(buf)
			if err != nil {
				if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
					log.Printf("Timeout reading ack packet: %v", err)
					continue RETRY
				}
				log.Printf("Error unmarshaling ack packet: %v", err)
				return err
			}

			// Unmarshal ACK packet
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
				return fmt.Errorf("Error packet received: %v", errorPacket)
			default:
				log.Printf("Unknown packet received: %v", buf)
				return fmt.Errorf("Unknown packet received: %v", buf)
			}

			log.Printf("Max retries reached for: %s", h.Conn.RemoteAddr())
			return fmt.Errorf("Max retries reached for: %s", h.Conn.RemoteAddr())
		}

		return nil
	}

	log.Printf("[%s] file sent", h.Conn.RemoteAddr())
	transferSucessful <- true
	return nil
}
