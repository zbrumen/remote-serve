package protocol

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"time"
)

type MessageData struct {
	Id       string    `json:"id"`
	Data     []byte    `json:"data"`
	Deadline time.Time `json:"deadline"`
	Close    bool      `json:"close"`
}

type Message struct {
	Id   string      `json:"id"`
	Type string      `json:"type"`
	Data MessageData `json:"data"`
}

func NewMessage(typ string, data MessageData) Message {
	return Message{
		Id:   GenerateChars(16),
		Type: typ,
		Data: data,
	}
}

func GenerateChars(n int, encoding ...string) string {
	out := make([]byte, n)
	_, err := rand.Read(out)
	if err != nil {
		panic(err)
	}
	if len(encoding) == 0 {
		encoding = append(encoding, "base64.url")
	}
	for e := range encoding {
		switch encoding[e] {
		case "base64.url":
			out = []byte(base64.RawURLEncoding.EncodeToString(out))
		case "base64":
			out = []byte(base64.RawStdEncoding.EncodeToString(out))
		case "hex":
			out = []byte(hex.EncodeToString(out))
		}
	}
	return string(out[:n])
}
