package main

import (
	"bytes"
	"testing"
)

func TestPrintBuildInfo(t *testing.T) {
	previousVersion, previousDate, previousCommit := buildVersion, buildDate, buildCommit
	defer func() { buildVersion, buildDate, buildCommit = previousVersion, previousDate, previousCommit }()

	for _, tt := range []struct {
		name, version, date, commit, want string
	}{
		{"empty", "", "", "", "Build version: N/A\nBuild date: N/A\nBuild commit: N/A\n"},
		{"set", "1.2.3", "2026-11-12", "abc123", "Build version: 1.2.3\nBuild date: 2026-11-12\nBuild commit: abc123\n"},
		{"partial", "1.2.3", "", "abc123", "Build version: 1.2.3\nBuild date: N/A\nBuild commit: abc123\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			buildVersion, buildDate, buildCommit = tt.version, tt.date, tt.commit
			var output bytes.Buffer
			printBuildInfo(&output)
			if got := output.String(); got != tt.want {
				t.Errorf("printBuildInfo() = %q, want %q", got, tt.want)
			}
		})
	}
}
