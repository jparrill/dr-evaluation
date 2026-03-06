package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const reportsDir = "reports"

// ensureReportsDir creates the reports/ directory if it doesn't exist.
func ensureReportsDir() error {
	return os.MkdirAll(reportsDir, 0755)
}

// defaultReportPath builds a path like reports/<prefix>_<timestamp>.md
func defaultReportPath(prefix string) string {
	ts := time.Now().UTC().Format("20060102_150405")
	return filepath.Join(reportsDir, fmt.Sprintf("%s_%s.md", prefix, ts))
}

// writeReport ensures the parent directory exists and writes the content.
func writeReport(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}
