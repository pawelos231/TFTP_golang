package main

import (
	client "TFTP/client/package"
	"TFTP/packets"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	filename = flag.String("p", "server/test.png", "Payload to fetch")
	compress = flag.Bool("c", true, "Compress payload")
	mode     = flag.String("m", "octet", "Transfer mode")
	serverIP = flag.String("s", "127.0.0.1:69", "Server address")
)

const (
	opcodeRRQ   = 1
	opcodeWRQ   = 2
	opcodeDATA  = 3
	opcodeACK   = 4
	opcodeERROR = 5
)

func main() {
	flag.Parse()
	transferSuccessful := false

	// Create RRQ packet
	rrq := packets.ReadRequest{
		FileName: *filename,
		Mode:     *mode,
		Compress: *compress,
	}

	localConn, err := client.SendRequest(rrq, serverIP)
	if err != nil {
		log.Fatalf("Error sending RRQ: %v", err)
		return
	}
	


	// Open file for writing the received data
	outputFileName := strings.ReplaceAll(("received_" + rrq.RequestType() + *filename ), "/", "_")
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		log.Fatalf("Error creating output file '%s': %v", outputFileName, err)
		return
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
		localConn.SetReadDeadline(time.Now().Add(10 * time.Second)) 

		n, addr, err := localConn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		// Update serverDataAddr if it's the first data packet
		if serverDataAddr == nil {
			serverDataAddr = addr
			fmt.Printf("Server data address set to %s\n", serverDataAddr)
		}

		// Ensure packets are from the correct server data address
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
				fmt.Println("Error writing to file:", err)
				break
			}

			// Send ACK
			ack := packets.Ack{BlockNumber: dataPck.BlockNumber}
			ackData, err := ack.MarshalBinary()
			if err != nil {
				fmt.Println("Error while marshaling ACK packet:", err)
				break
			}

			_, err = localConn.WriteTo(ackData, serverDataAddr)
			if err != nil {
				fmt.Println("Error while sending ACK packet:", err)
				break
			}

			if n < 516 {
				fmt.Println("Transfer completed")
				transferSuccessful = true
				break
			}

		case opcodeERROR:
			errorPck := packets.Error{}
			err = errorPck.UnmarshalBinary(buffer[:n])
			if err != nil {
				fmt.Println("Error unmarshaling ERROR packet:", err)
			} else {
				fmt.Printf("Received ERROR: %s\n", errorPck.Message)
			}
			break

		default:
			fmt.Printf("Unknown opcode %d received\n", opcode)
		}

		if transferSuccessful || opcode == opcodeERROR || err != nil {
			break
		}
	}

	// Finalize the file based on transfer success
	if transferSuccessful {
		log.Printf("File '%s' received successfully.", outputFileName)
		err = outputFile.Close()
		if err != nil {
			log.Printf("Error closing file '%s': %v", outputFileName, err)
		}
		outputFile = nil
	} else {
		log.Printf("File transfer failed. Cleaning up '%s'.", outputFileName)
	}
}
