package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/alex-laycalvert/go-chat/transport"

	"github.com/google/uuid"
)

var (
	assignedNicknames map[string]struct{}
	clients           *[]net.Conn
	clientIds         map[string]string
	mutex             *sync.Mutex
	history           []transport.Message
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalf("Must provide port")
		os.Exit(1)
	}
	port := args[0]
	hostname := "localhost"
	address := hostname + ":" + port
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
	defer listener.Close()
	log.Printf("go-chat server listening at %v", address)

	assignedNicknames = make(map[string]struct{})
	clientIds = make(map[string]string)
	clients = &[]net.Conn{}
	mutex = &sync.Mutex{}
	history = make([]transport.Message, 0)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %v", err)
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	/// Message Structure
	///
	/// NICKNAME_REQUEST [NICKNAME_LEN]
	///  - Very first message sent from the client. Requests a nickname
	///  - All other bytes in the message are ignored
	///
	/// MESSAGE []
	///  - [ID_LEN]: UUID of client
	///  - [*]: Text message
	defer conn.Close()

	var clientId string
	var nickname string

	reader := bufio.NewReader(conn)

	// Nickname request
	buffer := make([]byte, transport.LenNickname)
	for {
		_, err := reader.Read(buffer)
		if err != nil {
			log.Printf("[EROR] %v", err)
			return
		}
		requestedNickname := transport.TrimBufToString(buffer)
		if requestedNickname[len(requestedNickname)-1] == '\n' {
			requestedNickname = requestedNickname[:len(requestedNickname)-1]
		}
		log.Printf("[NNRQ] Requesting \"%v\"", requestedNickname)
		mutex.Lock()
		if _, found := assignedNicknames[requestedNickname]; !found {
			log.Printf("[NNRQ] \"%v\" approved", requestedNickname)

			// Add nickname
			assignedNicknames[requestedNickname] = struct{}{}

			// Assign client an ID
			clientId = uuid.NewString()

			// Add client ID to ID -> nickname map
			clientIds[clientId] = requestedNickname
			nickname = requestedNickname

			// Add client to list of clients
			log.Printf("[NEWC] Adding %v (%v)", requestedNickname, clientId)
			*clients = append(*clients, conn)
			mutex.Unlock()

			// Give client their ID
			conn.Write([]byte(clientId))
			break
		}
		mutex.Unlock()
		log.Printf("[NNRQ] \"%v\" taken", requestedNickname)
		conn.Write([]byte("NICKNAME_TAKEN"))
	}

	for {
		buffer, err := transport.ReadFromStream(reader, transport.RspEOT)
		if err != nil {
			break
		}
		if len(buffer) == 0 {
			// Invalid message
			log.Printf("[EROR] %v (%v): Invalid or Empty Message", nickname, clientId)
			continue
		}
		message := transport.TrimBufToString(buffer)
		message = strings.TrimSpace(message)
		mutex.Lock()
		bytesToWrite := []byte(transport.RspMessage)
		zeroPadding := make([]byte, 16)
		copy(zeroPadding, nickname)
		bytesToWrite = append(bytesToWrite, zeroPadding...)
		bytesToWrite = append(bytesToWrite, []byte(message+string(transport.RspEOT))...)
		log.Printf("[MESG] %v: %v", nickname, message)
		history = append(history, transport.Message{
			Nickname: nickname,
			Body:     message,
		})
		for _, client := range *clients {
			if client == conn {
				continue
			}
			if _, err := client.Write(bytesToWrite); err != nil {
				log.Printf("[EROR] %v (%v): %v", nickname, clientId, err)
			}
		}
		mutex.Unlock()
	}

	log.Printf("[DISC] %v (%v)", nickname, clientId)
	mutex.Lock()
	var clientIndex int
	for i, client := range *clients {
		if client == conn {
			clientIndex = i
			continue
		}
		bytesToWrite := []byte(transport.RspDisconnect + nickname + string(transport.RspEOT))
		if _, err := client.Write(bytesToWrite); err != nil {
			log.Printf("[EROR] %v (%v): %v", nickname, clientId, err)
		}
	}
	*clients = append((*clients)[:clientIndex], (*clients)[clientIndex+1:]...)
	mutex.Unlock()
}
