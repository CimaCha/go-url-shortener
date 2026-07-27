package flag

import (
	"flag"
	"fmt"
)

type BasicShortAddress struct {
	URL string
}

func NewBasicShortAddress(url string) *BasicShortAddress {
	return &BasicShortAddress{URL: url}
}

func NewBasicShortAddressFlag(flagName string, flagUsage string) *BasicShortAddress {
	basicShortAddress := BasicShortAddress{
		URL: "http://localhost:8080",
	}
	flag.Var(&basicShortAddress, flagName, flagUsage)
	return &basicShortAddress
}

func (basicShortAddress *BasicShortAddress) String() string {
	return fmt.Sprintf("%s", basicShortAddress.URL)
}

func (basicShortAddress *BasicShortAddress) Set(value string) error {
	if value == "" {
		return fmt.Errorf("url should not be empty")
	}
	basicShortAddress.URL = value
	return nil
}
