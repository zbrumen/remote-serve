package protocol

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
)

type Addr struct {
	network, address string
}

func (a Addr) Network() string {
	return a.network
}

func (a Addr) String() string {
	return a.address
}

func (a Addr) Encode() string {
	return base64.StdEncoding.EncodeToString([]byte(a.network)) + "_" + base64.StdEncoding.EncodeToString([]byte(a.address))
}

func NewAddr(addr net.Addr) Addr {
	return Addr{
		network: addr.Network(),
		address: addr.String(),
	}
}

func DecodeAddr(str string) (Addr, error) {
	cache := strings.Split(str, "_")
	if len(cache) != 2 {
		return Addr{}, fmt.Errorf("no _ found in encoded address")
	}
	network, err := base64.StdEncoding.DecodeString(cache[0])
	if err != nil {
		return Addr{}, err
	}
	address, err := base64.StdEncoding.DecodeString(cache[1])
	if err != nil {
		return Addr{}, err
	}
	return Addr{
		network: string(network),
		address: string(address),
	}, nil
}
