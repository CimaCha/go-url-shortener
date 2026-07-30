package flag

import (
	stdflag "flag"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCLIFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantA     string
		wantB     string
		wantError string
	}{
		{name: "default addresses", wantA: "localhost:8080", wantB: "http://localhost:8080"},
		{name: "addresses from CLI", args: []string{"-a", "server:9090", "-b", "https://short.example"}, wantA: "server:9090", wantB: "https://short.example"},
		{name: "address without port", args: []string{"-a", "server"}, wantA: "server", wantB: "http://localhost:8080"},
		{name: "address with extra separator", args: []string{"-a", "server:9090:extra"}, wantA: "server:9090:extra", wantB: "http://localhost:8080"},
		{name: "empty base address", args: []string{"-b", ""}, wantA: "localhost:8080", wantB: "http://localhost:8080", wantError: "url should not be empty"},
	}

	originalCommandLine := stdflag.CommandLine
	t.Cleanup(func() { stdflag.CommandLine = originalCommandLine })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdflag.CommandLine = stdflag.NewFlagSet(t.Name(), stdflag.ContinueOnError)
			stdflag.CommandLine.SetOutput(io.Discard)
			serviceAddress := NewNetAddressFlag("a", "address of service", "localhost:8080")
			shortAddress := NewNetAddressFlag("b", "basic address for short URL", "http://localhost:8080")

			err := stdflag.CommandLine.Parse(tt.args)

			if tt.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantError)
			}
			assert.Equal(t, tt.wantA, serviceAddress.String())
			assert.Equal(t, tt.wantB, shortAddress.String())
		})
	}
}
