package main

import (
	"log"
	"os"

	"github.com/alex-laycalvert/go-chat/transport"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalf("[FATL] Must provide port")
	}
	hostname := "localhost"
	port := args[0]
	server, err := transport.NewServer(hostname, port)
	if err != nil {
		log.Fatalf("[FATL] %v", err)
	}
	server.Start()
}
