package config

import (
	"flag"
	"github.com/stretchr/testify/require"
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
		envSecret    string
		wantAddress  string
		wantBaseURL  string
		wantFilePath string
		wantDatabase string
		wantSecret   string
	}{
		{name: "defaults", wantAddress: "localhost:8080", wantBaseURL: "http://localhost:8080", wantSecret: "secret_key"},
		{name: "flags", args: []string{"-a", "cli:8080", "-b", "http://cli:8080", "-f", "/tmp/cli-storage.json", "-d", "postgres://cli", "-k", "cli-secret"}, wantAddress: "cli:8080", wantBaseURL: "http://cli:8080", wantFilePath: "/tmp/cli-storage.json", wantDatabase: "postgres://cli", wantSecret: "cli-secret"},
		{name: "environment overrides flags", args: []string{"-a", "cli:8080", "-b", "http://cli:8080", "-f", "/tmp/cli-storage.json", "-d", "postgres://cli", "-k", "cli-secret"}, envAddress: "env:9090", envBaseURL: "http://env:9090", envFilePath: "/tmp/env-storage.json", envDatabase: "postgres://env", envSecret: "env-secret", wantAddress: "env:9090", wantBaseURL: "http://env:9090", wantFilePath: "/tmp/env-storage.json", wantDatabase: "postgres://env", wantSecret: "env-secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateConfig(t, tt.args)
			t.Setenv("SERVER_ADDRESS", tt.envAddress)
			t.Setenv("BASE_URL", tt.envBaseURL)
			t.Setenv("FILE_STORAGE_PATH", tt.envFilePath)
			t.Setenv("DATABASE_DSN", tt.envDatabase)
			t.Setenv("SECRET_KEY", tt.envSecret)

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
			if cfg.SecretKey != tt.wantSecret {
				t.Errorf("SecretKey = %q, want %q", cfg.SecretKey, tt.wantSecret)
			}
		})
	}
}

func isolateConfig(t *testing.T, args []string) {
	t.Helper()
	originalFlags, originalArgs := flag.CommandLine, os.Args
	t.Cleanup(func() { flag.CommandLine, os.Args = originalFlags, originalArgs })
	flag.CommandLine = flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = append([]string{"shortener"}, args...)
	for _, name := range []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN", "SECRET_KEY", "MAX_SHORTEN_ATTEMPTS", "DELETE_BATCH_CONCURRENCY", "DELETE_BATCH_SIZE", "DELETION_TIMEOUT"} {
		t.Setenv(name, "")
	}
}

func TestNumericConfig(t *testing.T) {
	for _, field := range []struct {
		flag, env    string
		get          func(*Config) int
		defaultValue int
	}{
		{"-m", "MAX_SHORTEN_ATTEMPTS", func(c *Config) int { return c.MaxShortURLsAttempts }, 5},
		{"-c", "DELETE_BATCH_CONCURRENCY", func(c *Config) int { return c.DeleteBatchConcurrency }, 10},
		{"-db", "DELETE_BATCH_SIZE", func(c *Config) int { return c.DeleteBatchSize }, 100},
		{"-dt", "DELETION_TIMEOUT", func(c *Config) int { return c.DeletionTimeout }, 10},
	} {
		t.Run(field.env, func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				isolateConfig(t, nil)
				cfg, err := New()
				require.NoError(t, err)
				require.Equal(t, field.defaultValue, field.get(cfg))
			})
			t.Run("flag", func(t *testing.T) {
				isolateConfig(t, []string{field.flag, "3"})
				cfg, err := New()
				require.NoError(t, err)
				require.Equal(t, 3, field.get(cfg))
			})
			t.Run("environment overrides flag", func(t *testing.T) {
				isolateConfig(t, []string{field.flag, "3"})
				t.Setenv(field.env, "7")
				cfg, err := New()
				require.NoError(t, err)
				require.Equal(t, 7, field.get(cfg))
			})
			for _, value := range []string{"0", "-1", "invalid", "999999999999999999999999"} {
				t.Run("invalid environment "+value, func(t *testing.T) {
					isolateConfig(t, nil)
					t.Setenv(field.env, value)
					_, err := New()
					require.Error(t, err)
				})
			}
			for _, value := range []string{"0", "-1", "invalid", "999999999999999999999999"} {
				t.Run("invalid flag "+value, func(t *testing.T) {
					isolateConfig(t, []string{field.flag, value})
					_, err := New()
					require.Error(t, err)
				})
			}
		})
	}
	t.Run("duration overflow", func(t *testing.T) {
		isolateConfig(t, []string{"-dt", "9223372037"})
		_, err := New()
		require.Error(t, err)
	})
}
