package report

import (
	"fmt"
	"time"

	"dr-evaluation/pkg/velero"
)

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "N/A"
	}
	s := int(d.Seconds())
	if s >= 3600 {
		return fmt.Sprintf("%dh %dm %ds", s/3600, (s%3600)/60, s%60)
	}
	if s >= 60 {
		return fmt.Sprintf("%dm %ds", s/60, s%60)
	}
	return fmt.Sprintf("%ds", s)
}

func shortTimestamp(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.UTC().Format("01-02 15:04:05")
}

func shortName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	return name[:maxLen-3] + "..."
}

func formatNumber(n int) string {
	if n == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	// Add comma separators
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

func countPhases(backups []velero.BackupInfo) map[string]int {
	phases := make(map[string]int)
	for _, b := range backups {
		p := b.Phase
		if p == "" {
			p = "Unknown"
		}
		phases[p]++
	}
	return phases
}

func countRestorePhases(restores []velero.RestoreInfo) map[string]int {
	phases := make(map[string]int)
	for _, r := range restores {
		p := r.Phase
		if p == "" {
			p = "Unknown"
		}
		phases[p]++
	}
	return phases
}

func sumBackupErrors(backups []velero.BackupInfo) (int, int) {
	var errors, warnings int
	for _, b := range backups {
		errors += b.Errors
		warnings += b.Warnings
	}
	return errors, warnings
}

func sumRestoreErrors(restores []velero.RestoreInfo) (int, int) {
	var errors, warnings int
	for _, r := range restores {
		errors += r.Errors
		warnings += r.Warnings
	}
	return errors, warnings
}
