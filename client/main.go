package main

import (
	client "TFTP/client/package"
	"TFTP/packets"
	"flag"
	"log"
	"time"
)

var (
	filename = flag.String("p", "cos.txt", "Payload to fetch / send")
	compress = flag.Bool("c", false, "Compress payload")
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
	transferSuccessful := make(chan bool)

	// Create RRQ packet
	// rrq := packets.ReadRequest{
	// 	FileName: *filename,
	// 	Mode:     *mode,
	// 	Compress: *compress,
	// }

	// Create WRQ packet
	wrq := packets.WriteRequest{
		FileName: *filename,
		Mode:     *mode,
		Compress: *compress,
	}

	// localConn, err := client.SendRequest(rrq, serverIP)
	// if err != nil {
	// 	log.Fatalf("Error sending RRQ: %v", err)
	// 	return
	// }

	localConn, err := client.SendRequest(wrq, serverIP)
	if err != nil {
		log.Fatalf("Error sending WRQ: %v", err)
		return
	}

	// Create handler
	timeout := 10 * time.Second
	handler := client.NewHandler(localConn, timeout)
	// go handler.HandleReadRequest(filename, transferSuccessful)
	go handler.HandleWriteRequest(filename, transferSuccessful)

	select {
	case <-transferSuccessful:
		log.Println("Transfer successful")
	case <-time.After(handler.Deadline):
		log.Println("Transfer timed out")
	}

}
