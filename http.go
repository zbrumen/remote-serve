package remote_serve

import (
	"encoding/base64"
	"github.com/zbrumen/remote-serve/net"
	"net/http"
	"net/url"
	"strings"
)

func ListenAndServe(addr string, handler http.Handler) error {
	srvr := &http.Server{Addr: addr, Handler: handler}
	if strings.Contains(addr, "remote://") {
		u, err := url.Parse(addr)
		if err != nil {
			return err
		}
		secret, _ := u.User.Password()
		port := ":8080"
		if u.Path != "/" {
			temp, err := base64.URLEncoding.DecodeString(u.Path[1:])
			if err != nil {
				return err
			}
			port = string(temp)
		}
		client, err := net.NewClient("tcp", u.Host, u.User.Username(), secret, port)
		if err != nil {
			return err
		}
		return srvr.Serve(client)
	}
	return srvr.ListenAndServe()
}

func ListenAndServeTLS(addr string, handler http.Handler, certFile, keyFile string) error {
	srvr := &http.Server{Addr: addr, Handler: handler}
	if strings.Contains(addr, "remote://") {
		u, err := url.Parse(addr)
		if err != nil {
			return err
		}
		secret, _ := u.User.Password()
		port := ":8080"
		if u.Path != "/" {
			temp, err := base64.URLEncoding.DecodeString(u.Path[1:])
			if err != nil {
				return err
			}
			port = string(temp)
		}
		client, err := net.NewClient("tcp", u.Host, u.User.Username(), secret, port)
		if err != nil {
			return err
		}
		return srvr.ServeTLS(client, certFile, keyFile)
	}
	return srvr.ListenAndServeTLS(certFile, keyFile)
}
