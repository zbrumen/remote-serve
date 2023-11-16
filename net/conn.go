package net

import (
	"bytes"
	"context"
	"fmt"
	"github.com/zbrumen/remote-serve/protocol"
	"net"
	"sync"
	"time"
)

type clientConn struct {
	local  protocol.Addr
	remote protocol.Addr

	cancel context.CancelFunc
	close  sync.Once

	background context.Context

	responder protocol.Sender
	requestId string

	readStream *bytes.Buffer
}

func newClientConn(msg protocol.Message, sender protocol.Sender) (*clientConn, error) {
	remote, err := protocol.DecodeAddr(msg.Data.Id)
	if err != nil {
		return nil, err
	}
	local, err := protocol.DecodeAddr(string(msg.Data.Data))
	if err != nil {
		return nil, err
	}
	background, cancel := context.WithCancel(context.Background())
	return &clientConn{
		local:      local,
		remote:     remote,
		cancel:     cancel,
		close:      sync.Once{},
		background: background,
		responder:  sender,
		requestId:  msg.Id,
		readStream: bytes.NewBuffer(nil),
	}, nil
}

func (c *clientConn) Read(b []byte) (n int, err error) {
	n, err = c.readStream.Read(b)
	return n, err
}

func (c *clientConn) Write(b []byte) (n int, err error) {
	n = len(b)
	return n, c.responder.Send(c.background, protocol.NewMessage("write", protocol.MessageData{
		Id:   c.requestId,
		Data: b,
	}))
}

func (c *clientConn) Close() error {
	err := fmt.Errorf("already closed")
	c.close.Do(func() {
		err = c.responder.Send(c.background, protocol.NewMessage("close", protocol.MessageData{
			Id:    c.requestId,
			Close: true,
		}))
		if err == nil {
			c.cancel()
		} else {
			c.close = sync.Once{}
		}
	})
	return err
}

func (c *clientConn) LocalAddr() net.Addr {
	return c.local
}

func (c *clientConn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *clientConn) SetDeadline(t time.Time) error {
	return c.responder.Send(c.background, protocol.NewMessage("set_deadline", protocol.MessageData{
		Id:       c.requestId,
		Deadline: t,
	}))
}

func (c *clientConn) SetReadDeadline(t time.Time) error {
	return c.responder.Send(c.background, protocol.NewMessage("set_read_deadline", protocol.MessageData{
		Id:       c.requestId,
		Deadline: t,
	}))
}

func (c *clientConn) SetWriteDeadline(t time.Time) error {
	return c.responder.Send(c.background, protocol.NewMessage("set_write_deadline", protocol.MessageData{
		Id:       c.requestId,
		Deadline: t,
	}))
}

func (c *clientConn) handleMessages(msg protocol.Message) (closed bool) {
	switch msg.Type {
	case "close":
		_ = c.Close()
		return true
	case "write":
		_, err := c.readStream.Write(msg.Data.Data)
		if err != nil {
			_ = c.Close()
			return true
		}
		return false
	default:
		fmt.Println("client.Connection received unknown msg.Type: " + msg.Type)
		return false
	}
}

type serverConn struct {
	listener net.Listener

	name string

	conns map[string]net.Conn
	sync  sync.RWMutex

	clientRequests  protocol.Sender
	clientResponses protocol.Receiver

	background context.Context
	close      context.CancelFunc
}

func (s *serverConn) backend() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			_ = s.Close()
			return
		}
		msg := protocol.NewMessage("create", protocol.MessageData{
			Id:       protocol.NewAddr(conn.RemoteAddr()).Encode(),
			Data:     []byte(protocol.NewAddr(conn.LocalAddr()).Encode()),
			Deadline: time.Now(),
			Close:    false,
		})
		if s.clientRequests != nil {
			err = s.clientRequests.Send(context.Background(), msg)
		} else {
			err = fmt.Errorf("no client connected")
		}
		if err != nil {
			fmt.Println("remote-serve: NO CLIENT CONNECTED FOR NEW CONNECTION")
			_ = conn.Close()
		} else {
			s.conns[msg.Id] = conn
			go func() {
				cache := make([]byte, 1024)
				for {
					n, err := conn.Read(cache)
					if s.clientRequests != nil {
						if err != nil {
							if s.clientRequests.Send(context.Background(), protocol.NewMessage("close", protocol.MessageData{
								Id:    msg.Id,
								Close: true,
							})) != nil {
								s.Close()
							} else {
								s.sync.Lock()
								delete(s.conns, msg.Data.Id)
								s.sync.Unlock()
							}
							return
						} else {
							if s.clientRequests.Send(context.Background(), protocol.NewMessage("write", protocol.MessageData{
								Id:       msg.Id,
								Data:     cache[:n],
								Deadline: time.Time{},
								Close:    false,
							})) != nil {
								s.Close()
								return
							}
						}
					} else {
						return
					}
				}
			}()
		}
	}
}

func (s *serverConn) clientBackend() {
	for {
		for msg := range s.clientResponses.Receive() {
			if conn, ok := s.conns[msg.Data.Id]; ok {
				switch msg.Type {
				case "close":
					s.sync.Lock()
					delete(s.conns, msg.Data.Id)
					s.sync.Unlock()
					_ = conn.Close()
				case "write":
					_, err := conn.Write(msg.Data.Data)
					if err != nil {
						err = s.clientRequests.Send(context.Background(), protocol.NewMessage("close", protocol.MessageData{
							Id: msg.Data.Id,
						}))
						if err != nil {
							s.Close()
						} else {
							s.sync.Lock()
							delete(s.conns, msg.Data.Id)
							s.sync.Unlock()
							_ = conn.Close()
						}
					}
				case "set_deadline":
					err := conn.SetDeadline(msg.Data.Deadline)
					if err != nil {
						err = s.clientRequests.Send(context.Background(), protocol.NewMessage("close", protocol.MessageData{
							Id: msg.Data.Id,
						}))
						if err != nil {
							s.Close()
						} else {
							s.sync.Lock()
							delete(s.conns, msg.Data.Id)
							s.sync.Unlock()
							_ = conn.Close()
						}
					}
				case "set_read_deadline":
					err := conn.SetReadDeadline(msg.Data.Deadline)
					if err != nil {
						err = s.clientRequests.Send(context.Background(), protocol.NewMessage("close", protocol.MessageData{
							Id: msg.Data.Id,
						}))
						if err != nil {
							s.Close()
						} else {
							s.sync.Lock()
							delete(s.conns, msg.Data.Id)
							s.sync.Unlock()
							_ = conn.Close()
						}
					}
				case "set_write_deadline":
					err := conn.SetWriteDeadline(msg.Data.Deadline)
					if err != nil {
						err = s.clientRequests.Send(context.Background(), protocol.NewMessage("close", protocol.MessageData{
							Id: msg.Data.Id,
						}))
						if err != nil {
							s.Close()
						} else {
							s.sync.Lock()
							delete(s.conns, msg.Data.Id)
							s.sync.Unlock()
							_ = conn.Close()
						}
					}
				}
			}
		}
	}
}

func (s *serverConn) Context() context.Context {
	return s.background
}

func (s *serverConn) Close() error {
	s.sync.Lock()
	for _, v := range s.conns {
		_ = v.Close()
	}
	if s.clientRequests != nil {
		_ = s.clientRequests.Close()
		_ = s.clientResponses.Close()
	}
	s.conns = map[string]net.Conn{}
	s.sync.Unlock()
	s.close()
	return s.listener.Close()
}

func (s *serverConn) String() string {
	return s.name
}

func newServerConn(addr, name string, receiver protocol.Receiver, sender protocol.Sender) (*serverConn, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	background, cancel := context.WithCancel(context.Background())
	out := &serverConn{
		listener:        listener,
		name:            name + " -> " + addr,
		conns:           make(map[string]net.Conn),
		sync:            sync.RWMutex{},
		clientRequests:  sender,
		clientResponses: receiver,
		background:      background,
		close:           cancel,
	}
	go out.backend()
	go out.clientBackend()
	return out, nil
}
