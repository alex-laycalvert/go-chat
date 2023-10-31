package transport

import (
	"bufio"
	"net"
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
	message := Message{
		Type:     MsgNicknameRequest,
		Nickname: nickname,
	}
	if _, err := client.connection.Write(message.Bytes()); err != nil {
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

func (client *Client) StartReceiving(messageChannel chan Message) {
	reader := bufio.NewReader(client.connection)

	for {
		buffer, err := ReadFromStream(reader, RspEOT)
		if err != nil {
			if err.Error() == "EOF" {
				messageChannel <- Message{Type: MsgServerDisconnect}
				break
			} else {
				messageChannel <- Message{Type: MsgError, Body: err.Error()}
			}
		}
		message, err := parseMessage(buffer)
		if err != nil {
			// failed to parse buffer
			continue
		}
		messageChannel <- *message
	}
}

func (client *Client) Send(messageStr string) error {
	message := Message{
		Type: MsgClientMessage,
		Body: messageStr,
	}
	_, err := client.connection.Write(message.Bytes())
	return err
}
