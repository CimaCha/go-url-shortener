package flag

import (
	"flag"
	"fmt"
)

type NetAddress struct {
	URL string
}

func NewNetAddress(url string) *NetAddress {
	return &NetAddress{URL: url}
}

func NewNetAddressFlag(flagName string, flagUsage string, defaultURL string) *NetAddress {
	netAddress := NetAddress{
		URL: defaultURL,
	}
	flag.Var(&netAddress, flagName, flagUsage)
	return &netAddress
}

func (netAddress *NetAddress) String() string {
	return netAddress.URL
}

func (netAddress *NetAddress) Set(value string) error {
	if value == "" {
		return fmt.Errorf("url should not be empty")
	}
	netAddress.URL = value
	return nil
}
