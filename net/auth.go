package net

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/zbrumen/remote-serve/protocol"
	"io"
	"net"
	"strings"
	"time"
)

func _authWriteMessage(writer io.Writer, msg string) error {
	_, err := writer.Write([]byte(base64.StdEncoding.EncodeToString([]byte(msg)) + " "))
	return err
}

func _authReadMessage(reader io.ReadCloser) (string, error) {
	var temp []byte
	cache := make([]byte, 1)
	for {
		n, err := reader.Read(cache)
		if err != nil {
			return "", err
		}
		if n == 0 {
			_ = reader.Close()
			return "", fmt.Errorf("closed")
		}
		if cache[0] == ' ' {
			out, err := base64.StdEncoding.DecodeString(string(temp))
			return string(out), err
		}
		temp = append(temp, cache[0])
	}
}

func _authHashChallenge(challenge, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(challenge))
	return hex.EncodeToString(h.Sum(nil))
}

func serverSideAuth(cl net.Conn, auths map[string]string) (protocol.Receiver, protocol.Sender, string, string, error) {
	hello, err := _authReadMessage(cl)
	if err != nil {
		return nil, nil, "", "", err
	}
	temp := strings.Split(hello, ",")
	if len(temp) != 2 {
		return nil, nil, "", "", fmt.Errorf("incorrect hello")
	}
	key := temp[1]
	port := temp[0]
	if secret, ok := auths[key]; ok {
		challenge := fmt.Sprintf("%s:%s:%s", key, time.Now().String(), protocol.GenerateChars(32))
		if err = _authWriteMessage(cl, challenge); err != nil {
			return nil, nil, key, port, err
		}
		challengeResp, err := _authReadMessage(cl)
		if err != nil {
			return nil, nil, key, port, err
		}
		if _authHashChallenge(challenge, secret) == challengeResp {
			return protocol.NewReceiver(cl), protocol.NewSender(cl), key, port, _authWriteMessage(cl, strings.Join(temp, ","))
		} else {
			return nil, nil, key, port, fmt.Errorf("unauthorized")
		}
	}
	_ = cl.Close()
	return nil, nil, key, port, fmt.Errorf("no such key")
}

func clientSideAuth(cl net.Conn, key, secret, port string) (protocol.Receiver, protocol.Sender, error) {
	if err := _authWriteMessage(cl, port+","+key); err != nil {
		return nil, nil, err
	}
	challenge, err := _authReadMessage(cl)
	if err != nil {
		return nil, nil, err
	}
	if err = _authWriteMessage(cl, _authHashChallenge(challenge, secret)); err != nil {
		return nil, nil, err
	}
	_, err = _authReadMessage(cl)
	return protocol.NewReceiver(cl), protocol.NewSender(cl), err
}
