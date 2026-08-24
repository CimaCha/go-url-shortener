package config

import (
	"flag"
	"io"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	originalCommandLine, originalArgs := flag.CommandLine, os.Args
	t.Cleanup(func() {
		flag.CommandLine = originalCommandLine
		os.Args = originalArgs
	})

	tests := []struct {
		name         string
		args         []string
		envAddress   string
		envBaseURL   string
		envFilePath  string
		wantAddress  string
		wantBaseURL  string
		wantFilePath string
	}{
		{name: "defaults", wantAddress: "localhost:8080", wantBaseURL: "http://localhost:8080"},
		{name: "flags", args: []string{"-a", "cli:8080", "-b", "http://cli:8080", "-f", "/tmp/cli-storage.json"}, wantAddress: "cli:8080", wantBaseURL: "http://cli:8080", wantFilePath: "/tmp/cli-storage.json"},
		{name: "environment overrides flags", args: []string{"-a", "cli:8080", "-b", "http://cli:8080", "-f", "/tmp/cli-storage.json"}, envAddress: "env:9090", envBaseURL: "http://env:9090", envFilePath: "/tmp/env-storage.json", wantAddress: "env:9090", wantBaseURL: "http://env:9090", wantFilePath: "/tmp/env-storage.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)
			os.Args = append([]string{"shortener"}, tt.args...)
			t.Setenv("SERVER_ADDRESS", tt.envAddress)
			t.Setenv("BASE_URL", tt.envBaseURL)
			t.Setenv("FILE_STORAGE_PATH", tt.envFilePath)

			cfg, err := New()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Address != tt.wantAddress {
				t.Errorf("Address = %q, want %q", cfg.Address, tt.wantAddress)
			}
			if cfg.BasicShortenAddress != tt.wantBaseURL {
				t.Errorf("BasicShortenAddress = %q, want %q", cfg.BasicShortenAddress, tt.wantBaseURL)
			}
			if cfg.FilePath != tt.wantFilePath {
				t.Errorf("FilePath = %q, want %q", cfg.FilePath, tt.wantFilePath)
			}
		})
	}
}
