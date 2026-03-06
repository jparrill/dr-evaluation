package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"dr-evaluation/pkg/velero"
)

type AnalysisInput struct {
	StartDate    time.Time
	EndDate      time.Time
	HasStartDate bool
	Namespace    string
	Sample       int
	Backups      []velero.BackupInfo
	Restores     []velero.RestoreInfo
}

func GenerateAnalysis(input AnalysisInput) string {
	var backups []velero.BackupInfo
	var restores []velero.RestoreInfo

	if input.HasStartDate {
		backups = velero.FilterBackupsByTime(input.Backups, input.StartDate, input.EndDate)
		restores = velero.FilterRestoresByTime(input.Restores, input.StartDate, input.EndDate)
	} else {
		// No start date: use all data up to EndDate, sampling handles the rest
		epoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		backups = velero.FilterBackupsByTime(input.Backups, epoch, input.EndDate)
		restores = velero.FilterRestoresByTime(input.Restores, epoch, input.EndDate)
	}

	restoreMap := velero.BuildRestoreMap(restores)
	groups := velero.GroupBackupsByType(backups)

	var sb strings.Builder

	// Header
	sb.WriteString("# Velero Backup/Restore Report\n\n")
	sb.WriteString(fmt.Sprintf("**Report generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("| Field | Value |\n")
	sb.WriteString("|-------|-------|\n")
	if input.HasStartDate {
		sb.WriteString(fmt.Sprintf("| **Period** | `%s` to `%s` |\n", input.StartDate.UTC().Format(time.RFC3339), input.EndDate.UTC().Format(time.RFC3339)))
	} else {
		sb.WriteString(fmt.Sprintf("| **Period** | Last %d samples (up to `%s`) |\n", input.Sample, input.EndDate.UTC().Format(time.RFC3339)))
	}
	sb.WriteString(fmt.Sprintf("| **Namespace** | `%s` |\n", input.Namespace))
	sb.WriteString(fmt.Sprintf("| **Total backups available** | %d |\n", len(backups)))
	sb.WriteString(fmt.Sprintf("| **Total restores available** | %d |\n", len(restores)))
	sb.WriteString("\n")

	typeOrder := []velero.BackupType{velero.BackupTypeFVT, velero.BackupTypeDailyFull, velero.BackupTypeHCDaily, velero.BackupTypeOther}
	typeLabels := map[velero.BackupType]string{
		velero.BackupTypeFVT:       "FVT Backups",
		velero.BackupTypeDailyFull: "Scheduled Daily Full Backups",
		velero.BackupTypeHCDaily:   "HC Daily Backups",
		velero.BackupTypeOther:     "Other Backups",
	}

	for _, bt := range typeOrder {
		items := groups[bt]
		if len(items) == 0 {
			continue
		}
		sampled := velero.SampleLast(items, input.Sample)
		hasTTL := bt == velero.BackupTypeDailyFull || bt == velero.BackupTypeHCDaily

		sb.WriteString(fmt.Sprintf("## %s (showing %d of %d)\n\n", typeLabels[bt], len(sampled), len(items)))
		writeBackupTable(&sb, sampled, hasTTL)
		sb.WriteString("\n")

		// Corresponding restores for FVT
		if bt == velero.BackupTypeFVT {
			var sampledRestores []velero.RestoreInfo
			for _, b := range sampled {
				sampledRestores = append(sampledRestores, restoreMap[b.Name]...)
			}
			if len(sampledRestores) > 0 {
				sb.WriteString("## FVT Restores (corresponding to sampled backups)\n\n")
				writeRestoreTable(&sb, sampledRestores)
				sb.WriteString("\n")
			}
		}
	}

	// Summary
	sb.WriteString("---\n\n")
	sb.WriteString("## Summary\n\n")
	writePhaseSummary(&sb, backups, restores)
	writeErrorSummary(&sb, backups, restores)

	// Duration stats
	fvtBackups := groups[velero.BackupTypeFVT]
	if len(fvtBackups) > 0 {
		stats := velero.CalcBackupDurationStats(fvtBackups)
		if stats.Count > 0 {
			sb.WriteString(fmt.Sprintf("### FVT Backup Duration Stats (all %d in period)\n\n", stats.Count))
			writeDurationStatsTable(&sb, stats)
			sb.WriteString("\n")
		}
	}

	var allFVTRestores []velero.RestoreInfo
	for _, b := range fvtBackups {
		allFVTRestores = append(allFVTRestores, restoreMap[b.Name]...)
	}
	if len(allFVTRestores) > 0 {
		stats := velero.CalcRestoreDurationStats(allFVTRestores)
		if stats.Count > 0 {
			sb.WriteString(fmt.Sprintf("### FVT Restore Duration Stats (all %d in period)\n\n", stats.Count))
			writeDurationStatsTable(&sb, stats)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func writeBackupTable(sb *strings.Builder, backups []velero.BackupInfo, withTTL bool) {
	if withTTL {
		sb.WriteString("| # | Name | Start | End | Duration | Items | Warnings | Errors | Phase | TTL |\n")
		sb.WriteString("|---|------|-------|-----|----------|-------|----------|--------|-------|-----|\n")
	} else {
		sb.WriteString("| # | Name | Start | End | Duration | Items | Warnings | Errors | Phase |\n")
		sb.WriteString("|---|------|-------|-----|----------|-------|----------|--------|-------|\n")
	}
	for i, b := range backups {
		line := fmt.Sprintf("| %d | `%s` | %s | %s | **%s** | %s | %d | %d | %s |",
			i+1,
			shortName(b.Name, 45),
			shortTimestamp(b.StartTimestamp),
			shortTimestamp(b.CompletionTimestamp),
			formatDuration(b.Duration),
			formatNumber(b.ItemsBackedUp),
			b.Warnings,
			b.Errors,
			b.Phase,
		)
		if withTTL {
			line += fmt.Sprintf(" %s |", b.TTL)
		}
		sb.WriteString(line + "\n")
	}
}

func writeRestoreTable(sb *strings.Builder, restores []velero.RestoreInfo) {
	sb.WriteString("| # | Name | Start | End | Duration | Items | Warnings | Errors | Phase |\n")
	sb.WriteString("|---|------|-------|-----|----------|-------|----------|--------|-------|\n")
	for i, r := range restores {
		sb.WriteString(fmt.Sprintf("| %d | `%s` | %s | %s | **%s** | %s | %d | %d | %s |\n",
			i+1,
			shortName(r.Name, 45),
			shortTimestamp(r.StartTimestamp),
			shortTimestamp(r.CompletionTimestamp),
			formatDuration(r.Duration),
			formatNumber(r.ItemsRestored),
			r.Warnings,
			r.Errors,
			r.Phase,
		))
	}
}

func writePhaseSummary(sb *strings.Builder, backups []velero.BackupInfo, restores []velero.RestoreInfo) {
	bPhases := countPhases(backups)
	sb.WriteString("**Backup phases:**\n")
	keys := sortedKeys(bPhases)
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("- %s: %d\n", k, bPhases[k]))
	}
	sb.WriteString("\n")

	rPhases := countRestorePhases(restores)
	if len(rPhases) > 0 {
		sb.WriteString("**Restore phases:**\n")
		keys = sortedKeys(rPhases)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("- %s: %d\n", k, rPhases[k]))
		}
		sb.WriteString("\n")
	}
}

func writeErrorSummary(sb *strings.Builder, backups []velero.BackupInfo, restores []velero.RestoreInfo) {
	bErrors, bWarnings := sumBackupErrors(backups)
	rErrors, rWarnings := sumRestoreErrors(restores)
	sb.WriteString("| Metric | Backups | Restores |\n")
	sb.WriteString("|--------|---------|----------|\n")
	sb.WriteString(fmt.Sprintf("| **Errors** | %d | %d |\n", bErrors, rErrors))
	sb.WriteString(fmt.Sprintf("| **Warnings** | %d | %d |\n", bWarnings, rWarnings))
	sb.WriteString("\n")
}

func writeDurationStatsTable(sb *strings.Builder, stats velero.DurationStats) {
	sb.WriteString("| Stat | Value |\n")
	sb.WriteString("|------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Min | %s |\n", formatDuration(stats.Min)))
	sb.WriteString(fmt.Sprintf("| Max | %s |\n", formatDuration(stats.Max)))
	sb.WriteString(fmt.Sprintf("| Avg | %s |\n", formatDuration(stats.Avg)))
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
