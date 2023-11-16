package main

import (
	"flag"
	"github.com/zbrumen/remote-serve/net"
	"strings"
)

func main() {
	port := flag.String("port", ":4200", "Address where the remote-server listens to")
	rawAuths := flag.String("auths", "user:pass;guest:guest", "Authentication library")
	flag.Parse()
	auths := make(map[string]string)
	for _, u := range strings.Split(*rawAuths, ";") {
		cache := strings.Split(u, ":")
		if len(cache) != 2 {
			panic("incorrect auths")
		}
		auths[cache[0]] = cache[1]
	}
	_, err := net.NewServer(*port, auths)
	if err != nil {
		panic(err)
	}
}
