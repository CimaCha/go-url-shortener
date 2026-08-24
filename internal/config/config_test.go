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
		envDatabase  string
		wantAddress  string
		wantBaseURL  string
		wantFilePath string
		wantDatabase string
	}{
		{name: "defaults", wantAddress: "localhost:8080", wantBaseURL: "http://localhost:8080"},
		{name: "flags", args: []string{"-a", "cli:8080", "-b", "http://cli:8080", "-f", "/tmp/cli-storage.json", "-d", "postgres://cli"}, wantAddress: "cli:8080", wantBaseURL: "http://cli:8080", wantFilePath: "/tmp/cli-storage.json", wantDatabase: "postgres://cli"},
		{name: "environment overrides flags", args: []string{"-a", "cli:8080", "-b", "http://cli:8080", "-f", "/tmp/cli-storage.json", "-d", "postgres://cli"}, envAddress: "env:9090", envBaseURL: "http://env:9090", envFilePath: "/tmp/env-storage.json", envDatabase: "postgres://env", wantAddress: "env:9090", wantBaseURL: "http://env:9090", wantFilePath: "/tmp/env-storage.json", wantDatabase: "postgres://env"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)
			os.Args = append([]string{"shortener"}, tt.args...)
			t.Setenv("SERVER_ADDRESS", tt.envAddress)
			t.Setenv("BASE_URL", tt.envBaseURL)
			t.Setenv("FILE_STORAGE_PATH", tt.envFilePath)
			t.Setenv("DATABASE_DSN", tt.envDatabase)

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
			if cfg.DatabaseURL != tt.wantDatabase {
				t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, tt.wantDatabase)
			}
		})
	}
}
