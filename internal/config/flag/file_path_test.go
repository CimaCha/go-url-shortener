package flag

import (
	stdflag "flag"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilePathFlag(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		want      string
		wantError string
	}{
		{name: "default path", want: "./storage.json"},
		{name: "path from CLI", args: []string{"-f", "/tmp/shortener.json"}, want: "/tmp/shortener.json"},
		{name: "empty path", args: []string{"-f", ""}, want: "./storage.json", wantError: "file path should not be empty"},
	}

	originalCommandLine := stdflag.CommandLine
	t.Cleanup(func() { stdflag.CommandLine = originalCommandLine })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdflag.CommandLine = stdflag.NewFlagSet(t.Name(), stdflag.ContinueOnError)
			stdflag.CommandLine.SetOutput(io.Discard)
			filePath := NewFilePathFlag("f", "path to storage", "./storage.json")

			err := stdflag.CommandLine.Parse(tt.args)

			if tt.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantError)
			}
			assert.Equal(t, tt.want, filePath.String())
		})
	}
}
