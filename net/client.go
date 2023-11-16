package net

import (
	"fmt"
	"github.com/zbrumen/remote-serve/protocol"
	"net"
	"sync"
)

type Client struct {
	serverRequests  protocol.Receiver
	serverResponder protocol.Sender
	serverConn      net.Conn

	conns chan *clientConn

	connections map[string]*clientConn
	c_sync      sync.RWMutex
}

func (c *Client) backend() {
	for req := range c.serverRequests.Receive() {
		switch req.Type {
		case "create":
			conn, err := newClientConn(req, c.serverResponder)
			if err != nil {
				fmt.Println("remote-server-client: CONNECTION ERROR: " + err.Error())
			} else {
				c.c_sync.Lock()
				c.connections[req.Id] = conn
				c.c_sync.Unlock()
				c.conns <- conn
			}
		default:
			if req.Data.Id != "" {
				c.c_sync.Lock()
				if conn := c.connections[req.Data.Id]; conn != nil {
					closed := conn.handleMessages(req)
					if closed {
						go func() {
							_ = conn.Close()
						}()
						delete(c.connections, req.Data.Id)
					}
				}
				c.c_sync.Unlock()
			}
		}
	}
	close(c.conns)
}

func (c *Client) Accept() (net.Conn, error) {
	out, ok := <-c.conns
	if !ok {
		return nil, fmt.Errorf("closed")
	}
	return out, nil
}

func (c *Client) Close() error {
	return c.serverRequests.Close()
}

func (c *Client) Addr() net.Addr {
	return c.serverConn.RemoteAddr()
}

func NewClient(network, addr, key, secret, port string) (net.Listener, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	recv, resp, err := clientSideAuth(conn, key, secret, port)
	if err != nil {
		return nil, err
	}
	out := &Client{
		serverRequests:  recv,
		serverResponder: resp,
		serverConn:      conn,
		conns:           make(chan *clientConn, 8),
		connections:     make(map[string]*clientConn),
		c_sync:          sync.RWMutex{},
	}
	go out.backend()
	return out, nil
}
