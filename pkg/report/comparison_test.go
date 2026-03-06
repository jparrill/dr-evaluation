package report

import (
	"strings"
	"testing"
	"time"

	"dr-evaluation/pkg/velero"
)

func TestGenerateComparison(t *testing.T) {
	cutoff := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	preBackups := []velero.BackupInfo{
		makeBackup("pre1-bkp-fvt", velero.BackupTypeFVT, cutoff.Add(-48*time.Hour), 300*time.Second, 551, "Completed"),
		makeBackup("pre2-bkp-fvt", velero.BackupTypeFVT, cutoff.Add(-24*time.Hour), 280*time.Second, 551, "Completed"),
		makeBackup("pre-daily-20260303", velero.BackupTypeHCDaily, cutoff.Add(-36*time.Hour), 200*time.Second, 108, "PartiallyFailed"),
	}
	preBackups[2].Errors = 63

	postBackups := []velero.BackupInfo{
		makeBackup("post1-bkp-fvt", velero.BackupTypeFVT, cutoff.Add(1*time.Hour), 150*time.Second, 551, "Completed"),
		makeBackup("post2-bkp-fvt", velero.BackupTypeFVT, cutoff.Add(12*time.Hour), 100*time.Second, 551, "Completed"),
	}

	preRestores := []velero.RestoreInfo{
		makeRestore("pre1-restore", "pre1-bkp-fvt", cutoff.Add(-47*time.Hour), 28*time.Second, 549, 60),
		makeRestore("pre2-restore", "pre2-bkp-fvt", cutoff.Add(-23*time.Hour), 30*time.Second, 549, 70),
	}

	postRestores := []velero.RestoreInfo{
		makeRestore("post1-restore", "post1-bkp-fvt", cutoff.Add(2*time.Hour), 25*time.Second, 549, 55),
		makeRestore("post2-restore", "post2-bkp-fvt", cutoff.Add(13*time.Hour), 24*time.Second, 549, 58),
	}

	allBackups := append(preBackups, postBackups...)
	allRestores := append(preRestores, postRestores...)

	content := GenerateComparison(ComparisonInput{
		CutoffDate: cutoff,
		Namespace:  "openshift-adp",
		Sample:     5,
		Backups:    allBackups,
		Restores:   allRestores,
	})

	// Header
	if !strings.Contains(content, "# Velero Plugin Change - Performance Analysis") {
		t.Error("missing comparison report header")
	}

	// Pre/post periods
	if !strings.Contains(content, "Pre-change period") {
		t.Error("missing pre-change period")
	}
	if !strings.Contains(content, "Post-change period") {
		t.Error("missing post-change period")
	}

	// FVT comparison table
	if !strings.Contains(content, "## FVT Backups") {
		t.Error("missing FVT Backups section")
	}
	if !strings.Contains(content, "Pre-change") && !strings.Contains(content, "Post-change") {
		t.Error("missing pre/post columns in comparison table")
	}

	// Delta percentages
	if !strings.Contains(content, "%") {
		t.Error("missing delta percentages")
	}

	// Duration trend
	if !strings.Contains(content, "### Duration Trend") {
		t.Error("missing Duration Trend section")
	}
	if !strings.Contains(content, "PRE") || !strings.Contains(content, "POST") {
		t.Error("missing PRE/POST labels in trend")
	}
	if !strings.Contains(content, "<<<") {
		t.Error("missing <<< markers for post-change entries")
	}

	// HC Daily failures
	if !strings.Contains(content, "Pre-change HC Daily Failures") {
		t.Error("missing HC Daily failures section")
	}
	if !strings.Contains(content, "PartiallyFailed") {
		t.Error("missing PartiallyFailed in failures table")
	}

	// Restores comparison
	if !strings.Contains(content, "## FVT Restores") {
		t.Error("missing FVT Restores section")
	}

	// Key takeaways
	if !strings.Contains(content, "## Key Takeaways") {
		t.Error("missing Key Takeaways section")
	}
}

func TestGenerateComparisonNoPreData(t *testing.T) {
	cutoff := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	postBackups := []velero.BackupInfo{
		makeBackup("post1-bkp-fvt", velero.BackupTypeFVT, cutoff.Add(1*time.Hour), 100*time.Second, 551, "Completed"),
	}

	content := GenerateComparison(ComparisonInput{
		CutoffDate: cutoff,
		Namespace:  "openshift-adp",
		Sample:     5,
		Backups:    postBackups,
		Restores:   nil,
	})

	if !strings.Contains(content, "Pre-change backups** | 0") {
		t.Error("should show 0 pre-change backups")
	}
	if !strings.Contains(content, "Post-change backups** | 1") {
		t.Error("should show 1 post-change backup")
	}
}

func TestGenerateComparisonEmptyData(t *testing.T) {
	cutoff := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	content := GenerateComparison(ComparisonInput{
		CutoffDate: cutoff,
		Namespace:  "openshift-adp",
		Sample:     5,
		Backups:    nil,
		Restores:   nil,
	})

	if !strings.Contains(content, "# Velero Plugin Change - Performance Analysis") {
		t.Error("should still generate header with empty data")
	}
	if !strings.Contains(content, "## Key Takeaways") {
		t.Error("should still generate takeaways section")
	}
}

func TestGenerateComparisonDeltaCalculation(t *testing.T) {
	cutoff := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	// Pre: avg 300s, Post: avg 150s => ~-50% improvement
	preBackups := []velero.BackupInfo{
		makeBackup("pre-bkp-fvt", velero.BackupTypeFVT, cutoff.Add(-1*time.Hour), 300*time.Second, 551, "Completed"),
	}
	postBackups := []velero.BackupInfo{
		makeBackup("post-bkp-fvt", velero.BackupTypeFVT, cutoff.Add(1*time.Hour), 150*time.Second, 551, "Completed"),
	}

	content := GenerateComparison(ComparisonInput{
		CutoffDate: cutoff,
		Namespace:  "openshift-adp",
		Sample:     5,
		Backups:    append(preBackups, postBackups...),
		Restores:   nil,
	})

	if !strings.Contains(content, "-50.0%") {
		t.Error("expected -50.0% delta for duration halved")
	}
}

func TestGenerateComparisonSuccessRateImprovement(t *testing.T) {
	cutoff := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	preBackups := []velero.BackupInfo{
		makeBackup("pre1-daily-20260303", velero.BackupTypeHCDaily, cutoff.Add(-48*time.Hour), 200*time.Second, 551, "Completed"),
		makeBackup("pre2-daily-20260304", velero.BackupTypeHCDaily, cutoff.Add(-24*time.Hour), 100*time.Second, 108, "PartiallyFailed"),
	}
	preBackups[1].Errors = 63

	postBackups := []velero.BackupInfo{
		makeBackup("post1-daily-20260305", velero.BackupTypeHCDaily, cutoff.Add(1*time.Hour), 50*time.Second, 551, "Completed"),
	}

	content := GenerateComparison(ComparisonInput{
		CutoffDate: cutoff,
		Namespace:  "openshift-adp",
		Sample:     5,
		Backups:    append(preBackups, postBackups...),
		Restores:   nil,
	})

	// Pre success rate should be 50% (1/2)
	if !strings.Contains(content, "50.0% (1/2)") {
		t.Error("expected 50.0% pre-change success rate")
	}
	// Post should be 100%
	if !strings.Contains(content, "100.0% (1/1)") {
		t.Error("expected 100.0% post-change success rate")
	}
}
