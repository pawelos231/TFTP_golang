package client

import (
	"TFTP/packets"
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

func SendReadRequest(rrq packets.Request, serverIP *string) (*net.UDPConn, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", *serverIP)
	if err != nil {
		fmt.Println("Invalid server address:", err)
		return nil, err
	}

	localConn, err := net.ListenUDP("udp", nil) // nil means any available local address
	if err != nil {
		fmt.Println("Failed to set up local UDP connection:", err)
		return nil, err
	}

	//fmt.Println("Sending RRQ for file:", *filename, "with mode:", *mode, "and compress:", *compress)
	rrqData, err := rrq.MarshalBinary()
	if err != nil {
		fmt.Println("Error while marshaling RRQ:", err)
		return nil, err
	}

	// Send RRQ to server
	_, err = localConn.WriteTo(rrqData, serverAddr)
	if err != nil {
		fmt.Println("Error while sending RRQ:", err)
		return nil, err
	}
	return localConn, nil
}

type Handler struct {
	Conn     *net.UDPConn
	Req      packets.Request
	Deadline time.Duration
}

func NewHandler(conn *net.UDPConn, req packets.Request, deadline time.Duration) *Handler {
	return &Handler{
		Conn:     conn,
		Req:      req,
		Deadline: deadline,
	}
}

func (h *Handler) HandleReadRequest(filename *string, transferSucessful chan bool) error {
	// Open file for writing the received data
	outputFileName := "received_" + h.Req.RequestType() + *filename
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
