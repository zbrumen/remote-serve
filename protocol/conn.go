package protocol

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"sync"
)

type Sender interface {
	Send(ctx context.Context, msg Message) error
	Close() error
}

type Receiver interface {
	Receive() <-chan Message
	Close() error
}

type sender struct {
	conn   net.Conn
	sender sync.Mutex
}

func (c *sender) Send(ctx context.Context, msg Message) error {
	c.sender.Lock()
	cache, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	b64 := base64.StdEncoding.EncodeToString(cache) + "\n"
	_, err = c.conn.Write([]byte(b64))
	c.sender.Unlock()
	return err
}

func (c *sender) Close() error {
	return c.conn.Close()
}

func NewSender(conn net.Conn) Sender {
	return &sender{
		conn:   conn,
		sender: sync.Mutex{},
	}
}

type receiver struct {
	conn net.Conn

	recv chan Message
}

func (c *receiver) Receive() <-chan Message {
	return c.recv
}

func (c *receiver) Close() error {
	return c.conn.Close()
}

func (c *receiver) backend() {
	buffer := bufio.NewReader(c.conn)
	for {
		b64, err := buffer.ReadString('\n')
		if err != nil {
			_ = c.Close()
			return
		}
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			_ = c.Close()
			return
		}
		var message Message
		err = json.Unmarshal(raw, &message)
		if err != nil {
			_ = c.Close()
			return
		}
		c.recv <- message
	}
}

func NewReceiver(c net.Conn) Receiver {
	out := &receiver{
		conn: c,

		recv: make(chan Message, 1),
	}
	go out.backend()
	return out
}
