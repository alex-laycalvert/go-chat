package main

import (
	"bufio"
	"log"
	"os"

	"github.com/alex-laycalvert/go-chat/transport"
)

func main() {
	args := os.Args[1:]
	client := initializeClient(args)
	defer client.Close()
	_ = requestNickname(client)
	messageChannel := make(chan transport.Message)
	go client.StartReceiving(messageChannel)
	go processUserInput(client)
	processIncomingMessages(messageChannel)
}

func initializeClient(args []string) *transport.Client {
	if len(args) < 2 {
		log.Fatalf("Must provide hostname and port")
		os.Exit(1)
	}
	hostname := args[0]
	port := args[1]
	client, err := transport.NewClient(hostname, port)
	if err != nil {
		log.Printf("Failed to connect with %v:%v", hostname, port)
		os.Exit(1)
	}
	return client
}

func requestNickname(client *transport.Client) *transport.Profile {
	termReader := bufio.NewReader(os.Stdin)
	for {
		buffer, err := transport.ReadFromStream(termReader, transport.RspNewline)
		if err != nil {
			log.Printf("Error: %v", err)
			os.Exit(1)
		}
		nickname := string(buffer)
		profile, success := client.RequestNickname(nickname)
		if success {
			return profile
		}
	}
}

func processUserInput(client *transport.Client) {
	reader := bufio.NewReader(os.Stdin)
	for {
		buffer, err := transport.ReadFromStream(reader, transport.RspNewline)
		if err != nil {
			if err.Error() == "EOF" {
				log.Printf("Disconnecting...")
				os.Exit(0)
				break
			} else {
				log.Printf("Error: %v", err)
				continue
			}
		}
		message := string(buffer)
		err = client.Send(message)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
	}
}

func processIncomingMessages(messageChannel chan transport.Message) {
	for {
		select {
		case message := <-messageChannel:
			log.Printf("%v", message.String())
		}
	}
}
