package report

import (
	"testing"
	"time"

	"dr-evaluation/pkg/velero"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{"zero", 0, "N/A"},
		{"negative", -1 * time.Second, "N/A"},
		{"seconds only", 45 * time.Second, "45s"},
		{"minutes and seconds", 3*time.Minute + 4*time.Second, "3m 4s"},
		{"exact minutes", 2 * time.Minute, "2m 0s"},
		{"hours minutes seconds", 1*time.Hour + 30*time.Minute + 15*time.Second, "1h 30m 15s"},
		{"exact hour", 1 * time.Hour, "1h 0m 0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.input)
			if got != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestShortTimestamp(t *testing.T) {
	ts := time.Date(2026, 3, 5, 12, 27, 46, 0, time.UTC)
	got := shortTimestamp(ts)
	if got != "03-05 12:27:46" {
		t.Errorf("got %q, want %q", got, "03-05 12:27:46")
	}

	got = shortTimestamp(time.Time{})
	if got != "N/A" {
		t.Errorf("got %q for zero time, want %q", got, "N/A")
	}
}

func TestShortName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short enough", "backup-123", 20, "backup-123"},
		{"exact length", "12345", 5, "12345"},
		{"needs truncation", "very-long-backup-name-here", 15, "very-long-ba..."},
		{"min truncation", "abcdef", 5, "ab..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortName(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("shortName(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1,000"},
		{18956, "18,956"},
		{1000000, "1,000,000"},
		{100, "100"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatNumber(tt.input)
			if got != tt.expected {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCountPhases(t *testing.T) {
	backups := []velero.BackupInfo{
		{Phase: "Completed"},
		{Phase: "Completed"},
		{Phase: "PartiallyFailed"},
		{Phase: ""},
	}

	phases := countPhases(backups)
	if phases["Completed"] != 2 {
		t.Errorf("Completed = %d, want 2", phases["Completed"])
	}
	if phases["PartiallyFailed"] != 1 {
		t.Errorf("PartiallyFailed = %d, want 1", phases["PartiallyFailed"])
	}
	if phases["Unknown"] != 1 {
		t.Errorf("Unknown = %d, want 1", phases["Unknown"])
	}
}

func TestCountRestorePhases(t *testing.T) {
	restores := []velero.RestoreInfo{
		{Phase: "Completed"},
		{Phase: "Completed"},
		{Phase: ""},
	}

	phases := countRestorePhases(restores)
	if phases["Completed"] != 2 {
		t.Errorf("Completed = %d, want 2", phases["Completed"])
	}
	if phases["Unknown"] != 1 {
		t.Errorf("Unknown = %d, want 1", phases["Unknown"])
	}
}

func TestSumBackupErrors(t *testing.T) {
	backups := []velero.BackupInfo{
		{Errors: 5, Warnings: 10},
		{Errors: 3, Warnings: 20},
		{Errors: 0, Warnings: 0},
	}

	errors, warnings := sumBackupErrors(backups)
	if errors != 8 {
		t.Errorf("errors = %d, want 8", errors)
	}
	if warnings != 30 {
		t.Errorf("warnings = %d, want 30", warnings)
	}
}

func TestSumRestoreErrors(t *testing.T) {
	restores := []velero.RestoreInfo{
		{Errors: 0, Warnings: 55},
		{Errors: 0, Warnings: 66},
	}

	errors, warnings := sumRestoreErrors(restores)
	if errors != 0 {
		t.Errorf("errors = %d, want 0", errors)
	}
	if warnings != 121 {
		t.Errorf("warnings = %d, want 121", warnings)
	}
}

func TestSumBackupErrorsEmpty(t *testing.T) {
	errors, warnings := sumBackupErrors(nil)
	if errors != 0 || warnings != 0 {
		t.Errorf("expected 0/0 for nil, got %d/%d", errors, warnings)
	}
}
