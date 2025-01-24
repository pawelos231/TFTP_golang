package main

import (
	server "TFTP/server/package"
	"flag"
	"fmt"
	"time"
)

var (
	address = flag.String("a", "127.0.0.1:69", "Address to listen on")
	payload = flag.String("p", "server/test.pdf", "Payload to send")
)

func main() {
	flag.Parse()

	s := server.Server{
		Timeout: 10 * time.Second,
		Retries: 10,
	}

	err := s.ListenAndServe(*address)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}

}
