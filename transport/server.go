package transport

import (
	"bufio"
	"log"
	"net"
	"sync"

	"github.com/google/uuid"
)

type Server struct {
	address           *net.TCPAddr
	clients           *[]net.Conn
	clientIds         map[string]string
	assignedNicknames map[string]struct{}
	mutex             *sync.Mutex
	history           []Message
}

func NewServer(hostname string, port string) (*Server, error) {
	address, err := net.ResolveTCPAddr("tcp", hostname+":"+port)
	if err != nil {
		return nil, err
	}
	return &Server{
		address:           address,
		clients:           &[]net.Conn{},
		clientIds:         make(map[string]string),
		assignedNicknames: make(map[string]struct{}),
		history:           make([]Message, 0),
		mutex:             &sync.Mutex{},
	}, nil
}

func (server *Server) Start() {
	listener, err := net.Listen("tcp", server.address.String())
	if err != nil {
		log.Fatalf("[FATL] %v", err)
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %v", err)
		}

		go server.handleConnection(conn)
	}
}

func (server *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	var clientId string
	var nickname string
	reader := bufio.NewReader(conn)
	isConnected := true
	for isConnected {
		buffer, err := ReadFromStream(reader, RspEOT)
		if err != nil {
			server.history = append(server.history, Message{
				Type:     MsgDisconnect,
				Nickname: nickname,
			})
			break
		}
		message, err := parseMessage(buffer)
		if err != nil {
			log.Printf("[EROR] %v", err)
		}
		server.mutex.Lock()
		switch message.Type {
		case MsgNicknameRequest:
			if _, found := server.assignedNicknames[message.Nickname]; !found {
				log.Printf("[NNRQ] \"%v\" approved", message.Nickname)

				// Add nickname
				server.assignedNicknames[message.Nickname] = struct{}{}

				// Assign client an ID
				clientId = uuid.NewString()

				// Add client ID to ID -> nickname map
				server.clientIds[clientId] = message.Nickname
				nickname = message.Nickname

				// Add client to list of clients
				log.Printf("[NEWC] Adding %v (%v)", nickname, clientId)
				*server.clients = append(*server.clients, conn)
				message := Message{
					Type:     MsgNewClient,
					Nickname: nickname,
				}
				server.writeToAllClientsExcept(message.Bytes(), nil)
				server.history = append(server.history, message)

				// Give client their ID
				conn.Write([]byte(clientId))
				break
			}
			log.Printf("[NNRQ] \"%v\" taken", message.Nickname)
			conn.Write([]byte("NICKNAME_TAKEN"))
			break
		case MsgRejoin:
			clientNickname, found := server.clientIds[message.Body]
			if !found {
				msg := Message{
					Type: MsgError,
					Body: "ID not found",
				}
				conn.Write(msg.Bytes())
				break
			}
			clientId = message.Body
			nickname = clientNickname
			*server.clients = append(*server.clients, conn)
			message := Message{
				Type:     MsgNewClient,
				Nickname: nickname,
			}
			server.writeToAllClientsExcept(message.Bytes(), nil)
			server.history = append(server.history, message)
			break
		case MsgClientMessage:
			message.Type = MsgMessage
			message.Nickname = nickname
			server.writeToAllClientsExcept(message.Bytes(), conn)
			log.Printf("[MESG] %v: %v", message.Nickname, message.Body)
			break
		case MsgDisconnect:
			isConnected = false
			break
		}
		server.history = append(server.history, *message)
		server.mutex.Unlock()
	}

	log.Printf("[DISC] %v (%v)", nickname, clientId)
	server.mutex.Lock()
	message := Message{
		Type:     MsgDisconnect,
		Nickname: nickname,
	}
	var clientIndex int
	for i, client := range *server.clients {
		if client == conn {
			clientIndex = i
			continue
		}
	}
	*server.clients = append((*server.clients)[:clientIndex], (*server.clients)[clientIndex+1:]...)
	server.writeToAllClientsExcept(message.Bytes(), conn)
	server.mutex.Unlock()
}

func (server *Server) writeToAllClientsExcept(bytesToWrite []byte, conn net.Conn) {
	for _, client := range *server.clients {
		if client == conn {
			continue
		}
		if _, err := client.Write(bytesToWrite); err != nil {
			log.Printf("[EROR] %v", err)
		}
	}
}
