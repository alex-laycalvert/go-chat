package main

import (
	"bufio"
	"log"
	"net"
	"os"

	"github.com/alex-laycalvert/go-chat/transport"
)

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatalf("Must provide hostname and port")
		os.Exit(1)
	}
	hostname := args[0]
	port := args[1]
	address, err := net.ResolveTCPAddr("tcp", hostname+":"+port)
	if err != nil {
		log.Printf("Invalid hostname and/or port")
		os.Exit(1)
	}
	conn, err := net.DialTCP("tcp", nil, address)
	if err != nil {
		log.Printf("Failed to connect")
		os.Exit(1)
	}
	defer conn.Close()

	_, _, err = requestNickname(conn)
	if err != nil {
		log.Printf("Failed to request nickname")
	}

	go handleServerMessages(conn)

	reader := bufio.NewReader(os.Stdin)
	for {
		buffer, err := transport.ReadFromStream(reader, transport.RspNewline)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		message := string(buffer)
		bytesToWrite := []byte(message + string(transport.RspEOT))
		if _, err = conn.Write(bytesToWrite); err != nil {
			log.Printf("Error: %v", err)
		}
	}
}

func handleServerMessages(conn net.Conn) {
	reader := bufio.NewReader(conn)

	for {
		buffer, err := transport.ReadFromStream(reader, transport.RspEOT)
		if err != nil {
			if err.Error() == "EOF" {
				log.Printf("Server disconnected, please close and reconnect")
			} else {
				log.Printf("Error: %v", err)
			}
			return
		}
		messageType := string(buffer[:transport.LenMessageType])
		switch messageType {
		case transport.RspMessage:
			nickname := transport.TrimBufToString(buffer[transport.LenMessageType:transport.LenNickname])
			message := transport.TrimBufToString(buffer[transport.LenNickname+transport.LenMessageType:])
			log.Printf("%v: %v", nickname, message)
		case transport.RspDisconnect:
			nickname := transport.TrimBufToString(buffer[transport.LenMessageType:])
			log.Printf("%v disconnected.", nickname)
		case transport.RspError:
		default:
		}
	}
}

func requestNickname(conn net.Conn) (string, string, error) {
	serverReader := bufio.NewReader(conn)
	termReader := bufio.NewReader(os.Stdin)

	for {
		buffer, err := transport.ReadFromStream(termReader, transport.RspNewline)
		if err != nil {
			return "", "", err
		}
		nickname := string(buffer)
		_, err = conn.Write([]byte(nickname))
		if err != nil {
			return "", "", err
		}
		buffer = make([]byte, transport.LenID)
		_, err = serverReader.Read(buffer)
		if err != nil {
			return "", "", err
		}
		id := transport.TrimBufToString(buffer)
		if id == transport.RspNicknameTaken || len(id) != transport.LenID {
			log.Printf("Nickname Taken, try again")
			continue
		}
		return id, nickname, nil
	}
}
