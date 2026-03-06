package report

import (
	"strings"
	"testing"
	"time"

	"dr-evaluation/pkg/velero"
)

func makeBackup(name string, btype velero.BackupType, created time.Time, duration time.Duration, items int, phase string) velero.BackupInfo {
	return velero.BackupInfo{
		Name:                name,
		Type:                btype,
		CreationTimestamp:   created,
		StartTimestamp:      created,
		CompletionTimestamp: created.Add(duration),
		Duration:            duration,
		Phase:               phase,
		ItemsBackedUp:       items,
		TotalItems:          items,
		TTL:                 "24h0m0s",
	}
}

func makeRestore(name, backupName string, created time.Time, duration time.Duration, items, warnings int) velero.RestoreInfo {
	return velero.RestoreInfo{
		Name:                name,
		BackupName:          backupName,
		CreationTimestamp:   created,
		StartTimestamp:      created,
		CompletionTimestamp: created.Add(duration),
		Duration:            duration,
		Phase:               "Completed",
		ItemsRestored:       items,
		TotalItems:          items,
		Warnings:            warnings,
	}
}

func TestGenerateAnalysisWithDateRange(t *testing.T) {
	base := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	backups := []velero.BackupInfo{
		makeBackup("old-bkp-fvt", velero.BackupTypeFVT, base.Add(-48*time.Hour), 200*time.Second, 551, "Completed"),
		makeBackup("b1-bkp-fvt", velero.BackupTypeFVT, base.Add(1*time.Hour), 150*time.Second, 551, "Completed"),
		makeBackup("b2-bkp-fvt", velero.BackupTypeFVT, base.Add(6*time.Hour), 120*time.Second, 551, "Completed"),
		makeBackup("daily-full-backup-20260305", velero.BackupTypeDailyFull, base, 60*time.Second, 18000, "Completed"),
	}
	restores := []velero.RestoreInfo{
		makeRestore("b1-bkp-fvt-restore", "b1-bkp-fvt", base.Add(2*time.Hour), 25*time.Second, 549, 55),
		makeRestore("b2-bkp-fvt-restore", "b2-bkp-fvt", base.Add(7*time.Hour), 26*time.Second, 549, 60),
	}

	content := GenerateAnalysis(AnalysisInput{
		StartDate:    base,
		EndDate:      base.Add(24 * time.Hour),
		HasStartDate: true,
		Namespace:    "openshift-adp",
		Sample:       5,
		Backups:      backups,
		Restores:     restores,
	})

	// Should contain report header
	if !strings.Contains(content, "# Velero Backup/Restore Report") {
		t.Error("missing report header")
	}

	// Should filter out the old backup
	if strings.Contains(content, "old-bkp-fvt") {
		t.Error("old backup should be filtered out")
	}

	// Should include in-range backups
	if !strings.Contains(content, "b1-bkp-fvt") {
		t.Error("missing b1 backup")
	}
	if !strings.Contains(content, "b2-bkp-fvt") {
		t.Error("missing b2 backup")
	}

	// Should include daily full
	if !strings.Contains(content, "daily-full-backup-20260305") {
		t.Error("missing daily full backup")
	}

	// Should include restores
	if !strings.Contains(content, "FVT Restores") {
		t.Error("missing FVT Restores section")
	}

	// Should include summary
	if !strings.Contains(content, "## Summary") {
		t.Error("missing Summary section")
	}

	// Should include duration stats
	if !strings.Contains(content, "FVT Backup Duration Stats") {
		t.Error("missing FVT Backup Duration Stats")
	}
}

func TestGenerateAnalysisWithoutStartDate(t *testing.T) {
	base := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	var backups []velero.BackupInfo
	for i := 0; i < 20; i++ {
		backups = append(backups, makeBackup(
			"fvt-backup-bkp-fvt",
			velero.BackupTypeFVT,
			base.Add(time.Duration(i)*6*time.Hour),
			time.Duration(100+i*10)*time.Second,
			551,
			"Completed",
		))
	}

	content := GenerateAnalysis(AnalysisInput{
		EndDate:      base.Add(200 * time.Hour),
		HasStartDate: false,
		Namespace:    "openshift-adp",
		Sample:       3,
		Backups:      backups,
		Restores:     nil,
	})

	// Should show "Last 3 samples" in period
	if !strings.Contains(content, "Last 3 samples") {
		t.Error("missing 'Last 3 samples' in period field")
	}

	// Should show "showing 3 of 20"
	if !strings.Contains(content, "showing 3 of 20") {
		t.Error("missing 'showing 3 of 20'")
	}
}

func TestGenerateAnalysisEmptyData(t *testing.T) {
	content := GenerateAnalysis(AnalysisInput{
		StartDate:    time.Now().Add(-24 * time.Hour),
		EndDate:      time.Now(),
		HasStartDate: true,
		Namespace:    "openshift-adp",
		Sample:       5,
		Backups:      nil,
		Restores:     nil,
	})

	if !strings.Contains(content, "# Velero Backup/Restore Report") {
		t.Error("missing report header even with empty data")
	}
	if !strings.Contains(content, "Total backups available** | 0") {
		t.Error("should show 0 backups")
	}
}

func TestGenerateAnalysisSampling(t *testing.T) {
	base := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	backups := []velero.BackupInfo{
		makeBackup("b1-bkp-fvt", velero.BackupTypeFVT, base.Add(1*time.Hour), 100*time.Second, 551, "Completed"),
		makeBackup("b2-bkp-fvt", velero.BackupTypeFVT, base.Add(2*time.Hour), 110*time.Second, 551, "Completed"),
		makeBackup("b3-bkp-fvt", velero.BackupTypeFVT, base.Add(3*time.Hour), 120*time.Second, 551, "Completed"),
		makeBackup("b4-bkp-fvt", velero.BackupTypeFVT, base.Add(4*time.Hour), 130*time.Second, 551, "Completed"),
		makeBackup("b5-bkp-fvt", velero.BackupTypeFVT, base.Add(5*time.Hour), 140*time.Second, 551, "Completed"),
	}

	content := GenerateAnalysis(AnalysisInput{
		StartDate:    base,
		EndDate:      base.Add(24 * time.Hour),
		HasStartDate: true,
		Namespace:    "openshift-adp",
		Sample:       2,
		Backups:      backups,
		Restores:     nil,
	})

	// Should show "showing 2 of 5"
	if !strings.Contains(content, "showing 2 of 5") {
		t.Error("expected 'showing 2 of 5'")
	}

	// Should contain the last 2 (b4, b5), not the first ones
	if !strings.Contains(content, "b4-bkp-fvt") {
		t.Error("b4 should be in the sample")
	}
	if !strings.Contains(content, "b5-bkp-fvt") {
		t.Error("b5 should be in the sample")
	}

	// b1 should not appear in the table (only 2 sampled)
	lines := strings.Split(content, "\n")
	b1InTable := false
	for _, line := range lines {
		if strings.Contains(line, "b1-bkp-fvt") && strings.HasPrefix(strings.TrimSpace(line), "|") {
			b1InTable = true
		}
	}
	if b1InTable {
		t.Error("b1 should not be in the sampled table")
	}
}
