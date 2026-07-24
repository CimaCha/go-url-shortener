package flag

import (
	"flag"
	"fmt"
	"strings"
)

type NetAddress struct {
	host string
	port string
}

func NewNetAddress(host string, port string) *NetAddress {
	return &NetAddress{host: host, port: port}
}

func NewNetAddressFlag(flagName string, flagUsage string) *NetAddress {
	netAddress := NetAddress{
		host: "localhost",
		port: "8080",
	}
	flag.Var(&netAddress, flagName, flagUsage)
	return &netAddress
}

func (netAddress *NetAddress) String() string {
	return fmt.Sprintf("%s:%s", netAddress.host, netAddress.port)
}

func (netAddress *NetAddress) Set(value string) error {
	address := strings.Split(value, ":")
	if len(address) != 2 {
		return fmt.Errorf("invalid net address: %s", value)
	}
	netAddress.host = address[0]
	netAddress.port = address[1]
	return nil
}
