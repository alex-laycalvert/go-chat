package transport

import "bufio"

const (
	RspEOT           = byte(4)
	RspNewline       = byte(10)
	RspNicknameTaken = "NICKNAME_TAKEN"
	RspDisconnect    = "DISC"
	RspMessage       = "MESG"
	RspError         = "EROR"
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
