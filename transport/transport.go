package transport

import (
	"bufio"
	"net"
)

const (
	RspEOF           = byte(0)
	RspEOT           = byte(4)
	RspNewline       = byte(10)
	RspNicknameTaken = "NICKNAME_TAKEN"
	RspDisconnect    = "DISC"
	RspMessage       = "MESG"
	RspError         = "EROR"
	RspNewClient     = "NCLI"
	LenMessageType   = 4
	LenID            = 36
	LenNickname      = 16
)

type Message struct {
	Nickname string
	Body     string
}

// Converts the buffer to a string and removes all trailing EOT's and NULL characters
func TrimBufToString(buf []byte) string {
	if len(buf) == 0 {
		return ""
	}
	var end int
	for end = len(buf) - 1; end >= 0; end-- {
		if buf[end] != 0 && buf[end] != RspEOT {
			break
		}
	}
	return string(buf[:end+1])
}

func ReadFromStream(reader *bufio.Reader, delim byte) ([]byte, error) {
	buffer, err := reader.ReadBytes(delim)
	if err != nil {
		return nil, err
	}
	buffer = buffer[:len(buffer)-1]
	return buffer, nil
}

type Client struct {
	serverAddress *net.TCPAddr
	connection    *net.TCPConn
}

type Profile struct {
	ID       string
	Nickname string
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
