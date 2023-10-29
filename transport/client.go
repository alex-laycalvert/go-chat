package transport

import (
	"net"
	"bufio"
)

type Client struct {
	serverAddress *net.TCPAddr
	connection    *net.TCPConn
}

func NewClient(hostname string, port string) (*Client, error) {
	address, err := net.ResolveTCPAddr("tcp", hostname+":"+port)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, address)
	if err != nil {
		return nil, err
	}
	return &Client{
		serverAddress: address,
		connection:    conn,
	}, nil
}

func (client *Client) Close() {
	client.Close()
}

func (client *Client) RequestNickname(nickname string) (*Profile, bool) {
	reader := bufio.NewReader(client.connection)
	if _, err := client.connection.Write([]byte(nickname)); err != nil {
		return nil, false
	}
	buffer := make([]byte, LenID)
	if _, err := reader.Read(buffer); err != nil {
		return nil, false
	}
	id := TrimBufToString(buffer)
	if id == RspNicknameTaken || len(id) != LenID {
		return nil, false
	}
	return &Profile{
		ID:       id,
		Nickname: nickname,
	}, true
}

func (client *Client) StartReceiving(messageChannel chan string) {
	reader := bufio.NewReader(client.connection)

	for {
		buffer, err := ReadFromStream(reader, RspEOT)
		if err != nil {
			if err.Error() == "EOF" {
				messageChannel <- "Server disconnected, please close and reconnect."
				break
			} else {
				messageChannel <- "[EROR] " + err.Error()
			}
		}
		messageType := string(buffer[:LenMessageType])
		switch messageType {
		case RspMessage:
			nickname := TrimBufToString(buffer[LenMessageType:LenNickname])
			message := TrimBufToString(buffer[LenNickname+LenMessageType:])
			messageChannel <- nickname + ": " + message
			continue
		case RspDisconnect:
			nickname := TrimBufToString(buffer[LenMessageType:])
			messageChannel <- nickname + " disconnected."
			continue
		case RspError:
			err := TrimBufToString(buffer[LenMessageType:])
			messageChannel <- "[EROR] " + err
			continue
		case RspNewClient:
			nickname := TrimBufToString(buffer[LenMessageType:])
			messageChannel <- nickname + " just joined."
		default:
			continue
		}
	}
}

func (client *Client) Send(message string) error {
	bytesToWrite := []byte(message + string(RspEOT))
	_, err := client.connection.Write(bytesToWrite)
	return err
}
