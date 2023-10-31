package transport

import (
	"errors"
)

type MsgType uint8

const (
	MsgNewClient MsgType = iota
	MsgMessage
	MsgClientMessage
	MsgDisconnect
	MsgServerDisconnect
	MsgError
	MsgNicknameRequest
	MsgRejoin
)

type Message struct {
	Type     MsgType
	Nickname string
	Body     string
}

func (message *Message) Bytes() []byte {
	bytesToWrite := make([]byte, 0)
	bytesToWrite = append(bytesToWrite, byte(message.Type))
	switch message.Type {
	case MsgNewClient, MsgDisconnect, MsgNicknameRequest:
		bytesToWrite = append(bytesToWrite, padNicknameBytes(message.Nickname)...)
		break
	case MsgMessage:
		bytesToWrite = append(bytesToWrite, padNicknameBytes(message.Nickname)...)
		bytesToWrite = append(bytesToWrite, []byte(message.Body)...)
		break
	case MsgClientMessage, MsgError, MsgRejoin:
		bytesToWrite = append(bytesToWrite, []byte(message.Body)...)
		break
	case MsgServerDisconnect:
		break
	}
	bytesToWrite = append(bytesToWrite, RspEOT)
	return bytesToWrite
}

func (message *Message) String() string {
	switch message.Type {
	case MsgNewClient:
		return message.Nickname + " just joined."
	case MsgMessage:
		return message.Nickname + ": " + message.Body
	case MsgClientMessage, MsgError, MsgRejoin:
		return message.Body
	case MsgDisconnect:
		return message.Nickname + " left."
	case MsgServerDisconnect:
		return "Disconnected from server."
	case MsgNicknameRequest:
		return "Requesting " + message.Nickname
	default:
		return ""
	}
}

func padNicknameBytes(nickname string) []byte {
	zeroPadding := make([]byte, LenNickname)
	copy(zeroPadding, []byte(nickname))
	return zeroPadding
}

// Parses the bytes into a Message
func parseMessage(data []byte) (*Message, error) {
	if len(data) == 0 {
		return nil, errors.New("Cannot parse message from empty array")
	}
	msgType := MsgType(data[0])
	if !isValidMsgType(msgType) {
		return nil, errors.New("Invalid message type " + string(msgType))
	}
	message := Message{Type: msgType}

	switch msgType {
	case MsgNewClient, MsgDisconnect, MsgNicknameRequest:
		message.Nickname = TrimBufToString(data[1:LenNickname])
		break
	case MsgMessage:
		message.Nickname = TrimBufToString(data[1:LenNickname])
		message.Body = TrimBufToString(data[LenNickname:])
		break
	case MsgClientMessage, MsgError, MsgRejoin:
		message.Body = TrimBufToString(data[1:])
		break
	}
	return &message, nil
}

func isValidMsgType(val MsgType) bool {
	switch val {
	case MsgNewClient, MsgMessage, MsgClientMessage, MsgDisconnect, MsgServerDisconnect, MsgError, MsgNicknameRequest, MsgRejoin:
		return true
	default:
		return false
	}
}
