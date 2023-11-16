package net

import (
	"fmt"
	"net"
	"sync"
)

type Server struct {
	comLinkServer net.Listener
	auth          map[string]string

	conns map[string]*serverConn
	sync  sync.RWMutex
}

func (s *Server) clientsBackend() {
	for {
		client, err := s.comLinkServer.Accept()
		if err != nil {
			fmt.Println("remote-serve: SERVER IS CLOSING")
			s.sync.Lock()
			for _, v := range s.conns {
				_ = v.Close()
				fmt.Println("remote-serve: CLOSING " + v.String() + " SERVER")
			}
			s.sync.Unlock()
			return
		}
		receiver, sender, name, port, err := serverSideAuth(client, s.auth)
		if err != nil {
			fmt.Println("remote-serve: CLIENT_AUTH ERROR: " + err.Error())
		} else {
			// break previous connections and reestablish
			s.sync.Lock()
			if c, ok := s.conns[port]; ok {
				_ = c.Close()
				delete(s.conns, port)
			}
			conn, err := newServerConn(port, name, receiver, sender)
			if err == nil {
				s.conns[port] = conn
				go func() {
					<-conn.Context().Done()
					s.sync.Lock()
					delete(s.conns, port)
					s.sync.Unlock()
					fmt.Println("remote-serve: SERVER " + port + " IS CLOSING")
				}()
			} else {
				fmt.Println("remote-serve: CANNOT CREATE SERVER ERROR: " + err.Error())
				_ = client.Close()
			}
			s.sync.Unlock()
		}
	}
}

func NewServer(addr string, auth map[string]string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	out := &Server{
		comLinkServer: listener,
		auth:          auth,
		conns:         make(map[string]*serverConn),
		sync:          sync.RWMutex{},
	}
	go out.clientsBackend()
	return out, nil
}
