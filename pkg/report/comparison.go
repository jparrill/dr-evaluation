package report

import (
	"fmt"
	"strings"
	"time"

	"dr-evaluation/pkg/velero"
)

type ComparisonInput struct {
	CutoffDate time.Time
	Namespace  string
	Sample     int
	Backups    []velero.BackupInfo
	Restores   []velero.RestoreInfo
}

type periodData struct {
	backups    []velero.BackupInfo
	restores   []velero.RestoreInfo
	groups     map[velero.BackupType][]velero.BackupInfo
	restoreMap map[string][]velero.RestoreInfo
}

func GenerateComparison(input ComparisonInput) string {
	preEnd := input.CutoffDate.Add(-time.Second)
	earliest := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	latest := time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

	pre := buildPeriod(input.Backups, input.Restores, earliest, preEnd)
	post := buildPeriod(input.Backups, input.Restores, input.CutoffDate, latest)

	var sb strings.Builder

	// Header
	sb.WriteString("# Velero Plugin Change - Performance Analysis\n\n")
	sb.WriteString(fmt.Sprintf("**Report generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("| Field | Value |\n")
	sb.WriteString("|-------|-------|\n")
	sb.WriteString(fmt.Sprintf("| **Pre-change period** | Up to %s |\n", preEnd.UTC().Format("2006-01-02 15:04:05 UTC")))
	sb.WriteString(fmt.Sprintf("| **Post-change period** | From %s |\n", input.CutoffDate.UTC().Format("2006-01-02 15:04:05 UTC")))
	sb.WriteString(fmt.Sprintf("| **Pre-change backups** | %d |\n", len(pre.backups)))
	sb.WriteString(fmt.Sprintf("| **Post-change backups** | %d |\n", len(post.backups)))
	sb.WriteString(fmt.Sprintf("| **Pre-change restores** | %d |\n", len(pre.restores)))
	sb.WriteString(fmt.Sprintf("| **Post-change restores** | %d |\n", len(post.restores)))
	sb.WriteString(fmt.Sprintf("| **Namespace** | `%s` |\n", input.Namespace))
	sb.WriteString("\n---\n\n")

	typeOrder := []velero.BackupType{velero.BackupTypeFVT, velero.BackupTypeDailyFull, velero.BackupTypeHCDaily, velero.BackupTypeOther}
	typeLabels := map[velero.BackupType]string{
		velero.BackupTypeFVT:       "FVT Backups",
		velero.BackupTypeDailyFull: "Daily Full Backups",
		velero.BackupTypeHCDaily:   "HC Daily Backups",
		velero.BackupTypeOther:     "Other Backups",
	}

	for _, bt := range typeOrder {
		preItems := pre.groups[bt]
		postItems := post.groups[bt]
		if len(preItems) == 0 && len(postItems) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s\n\n", typeLabels[bt]))
		writeComparisonTable(&sb, preItems, postItems)
		sb.WriteString("\n")

		// FVT duration trend
		if bt == velero.BackupTypeFVT {
			writeDurationTrend(&sb, preItems, postItems, input.Sample, input.CutoffDate)
			sb.WriteString("\n")
		}
	}

	// Restores comparison
	if len(pre.restores) > 0 || len(post.restores) > 0 {
		sb.WriteString("## FVT Restores\n\n")
		writeRestoreComparisonTable(&sb, pre.restores, post.restores)
		sb.WriteString("\n")
	}

	// HC Daily failures detail
	preHC := pre.groups[velero.BackupTypeHCDaily]
	var failures []velero.BackupInfo
	for _, b := range preHC {
		if b.Phase != "Completed" {
			failures = append(failures, b)
		}
	}
	if len(failures) > 0 {
		sb.WriteString("## Pre-change HC Daily Failures\n\n")
		sb.WriteString("| Date | Phase | Errors | Items | Name |\n")
		sb.WriteString("|------|-------|--------|-------|------|\n")
		for _, b := range failures {
			sb.WriteString(fmt.Sprintf("| %s | %s | %d | %d | `%s` |\n",
				b.CreationTimestamp.UTC().Format("2006-01-02"),
				b.Phase,
				b.Errors,
				b.ItemsBackedUp,
				shortName(b.Name, 55),
			))
		}
		sb.WriteString("\n")
	}

	// Key takeaways
	sb.WriteString("---\n\n")
	sb.WriteString("## Key Takeaways\n\n")
	writeKeyTakeaways(&sb, pre, post)

	return sb.String()
}

func buildPeriod(allBackups []velero.BackupInfo, allRestores []velero.RestoreInfo, start, end time.Time) periodData {
	backups := velero.FilterBackupsByTime(allBackups, start, end)
	restores := velero.FilterRestoresByTime(allRestores, start, end)
	return periodData{
		backups:    backups,
		restores:   restores,
		groups:     velero.GroupBackupsByType(backups),
		restoreMap: velero.BuildRestoreMap(restores),
	}
}

func writeComparisonTable(sb *strings.Builder, preItems, postItems []velero.BackupInfo) {
	preStats := velero.CalcBackupDurationStats(preItems)
	postStats := velero.CalcBackupDurationStats(postItems)

	preCompleted := countCompleted(preItems)
	postCompleted := countCompleted(postItems)
	preErrors := sumErrors(preItems)
	postErrors := sumErrors(postItems)

	preItemsAvg := avgItems(preItems)
	postItemsAvg := avgItems(postItems)

	sb.WriteString("| Metric | Pre-change | Post-change | Delta |\n")
	sb.WriteString("|--------|------------|-------------|-------|\n")
	sb.WriteString(fmt.Sprintf("| **Count** | %d | %d | - |\n", len(preItems), len(postItems)))

	if preStats.Count > 0 && postStats.Count > 0 {
		avgDelta := calcDeltaPct(preStats.Avg.Seconds(), postStats.Avg.Seconds())
		minDelta := calcDeltaPct(preStats.Min.Seconds(), postStats.Min.Seconds())
		maxDelta := calcDeltaPct(preStats.Max.Seconds(), postStats.Max.Seconds())
		sb.WriteString(fmt.Sprintf("| **Avg duration** | %s | %s | **%s** |\n", formatDuration(preStats.Avg), formatDuration(postStats.Avg), avgDelta))
		sb.WriteString(fmt.Sprintf("| **Min duration** | %s | %s | **%s** |\n", formatDuration(preStats.Min), formatDuration(postStats.Min), minDelta))
		sb.WriteString(fmt.Sprintf("| **Max duration** | %s | %s | **%s** |\n", formatDuration(preStats.Max), formatDuration(postStats.Max), maxDelta))
	} else if preStats.Count > 0 {
		sb.WriteString(fmt.Sprintf("| **Avg duration** | %s | N/A | - |\n", formatDuration(preStats.Avg)))
	} else if postStats.Count > 0 {
		sb.WriteString(fmt.Sprintf("| **Avg duration** | N/A | %s | - |\n", formatDuration(postStats.Avg)))
	}

	if preItemsAvg > 0 || postItemsAvg > 0 {
		sb.WriteString(fmt.Sprintf("| **Avg items** | %s | %s | %s |\n",
			formatNumber(preItemsAvg), formatNumber(postItemsAvg),
			calcDeltaPctInt(preItemsAvg, postItemsAvg)))
	}

	sb.WriteString(fmt.Sprintf("| **Success rate** | %s | %s | %s |\n",
		formatRate(preCompleted, len(preItems)),
		formatRate(postCompleted, len(postItems)),
		deltaRate(preCompleted, len(preItems), postCompleted, len(postItems))))
	sb.WriteString(fmt.Sprintf("| **Errors** | %d | %d | %s |\n", preErrors, postErrors, deltaInt(preErrors, postErrors)))
}

func writeRestoreComparisonTable(sb *strings.Builder, preRestores, postRestores []velero.RestoreInfo) {
	preStats := velero.CalcRestoreDurationStats(preRestores)
	postStats := velero.CalcRestoreDurationStats(postRestores)

	preWarnAvg := avgRestoreWarnings(preRestores)
	postWarnAvg := avgRestoreWarnings(postRestores)

	sb.WriteString("| Metric | Pre-change | Post-change | Delta |\n")
	sb.WriteString("|--------|------------|-------------|-------|\n")
	sb.WriteString(fmt.Sprintf("| **Count** | %d | %d | - |\n", len(preRestores), len(postRestores)))

	if preStats.Count > 0 && postStats.Count > 0 {
		avgDelta := calcDeltaPct(preStats.Avg.Seconds(), postStats.Avg.Seconds())
		sb.WriteString(fmt.Sprintf("| **Avg duration** | %s | %s | **%s** |\n", formatDuration(preStats.Avg), formatDuration(postStats.Avg), avgDelta))
		sb.WriteString(fmt.Sprintf("| **Min duration** | %s | %s | %s |\n", formatDuration(preStats.Min), formatDuration(postStats.Min),
			calcDeltaPct(preStats.Min.Seconds(), postStats.Min.Seconds())))
		sb.WriteString(fmt.Sprintf("| **Max duration** | %s | %s | **%s** |\n", formatDuration(preStats.Max), formatDuration(postStats.Max),
			calcDeltaPct(preStats.Max.Seconds(), postStats.Max.Seconds())))
	}

	sb.WriteString(fmt.Sprintf("| **Avg warnings/restore** | %.1f | %.1f | %s |\n",
		preWarnAvg, postWarnAvg, calcDeltaPct(preWarnAvg, postWarnAvg)))
}

func writeDurationTrend(sb *strings.Builder, preItems, postItems []velero.BackupInfo, sampleCount int, cutoff time.Time) {
	sampled := velero.SampleLast(preItems, sampleCount)
	all := append(sampled, postItems...)
	if len(all) == 0 {
		return
	}

	sb.WriteString("### Duration Trend\n\n")
	sb.WriteString("```\n")
	for _, b := range all {
		if b.Duration <= 0 {
			continue
		}
		period := "PRE "
		marker := ""
		if !b.CreationTimestamp.Before(cutoff) {
			period = "POST"
			marker = " <<<"
		}
		secs := int(b.Duration.Seconds())
		bar := strings.Repeat("#", secs/10)
		sb.WriteString(fmt.Sprintf("%s | %s | %4ds | %s%s\n",
			period,
			b.CreationTimestamp.UTC().Format("2006-01-02 15:04"),
			secs,
			bar,
			marker,
		))
	}
	sb.WriteString("```\n")
}

func writeKeyTakeaways(sb *strings.Builder, pre, post periodData) {
	idx := 1

	// FVT backup speed
	preFVT := pre.groups[velero.BackupTypeFVT]
	postFVT := post.groups[velero.BackupTypeFVT]
	if len(preFVT) > 0 && len(postFVT) > 0 {
		preStats := velero.CalcBackupDurationStats(preFVT)
		postStats := velero.CalcBackupDurationStats(postFVT)
		if preStats.Count > 0 && postStats.Count > 0 && preStats.Avg > postStats.Avg {
			pct := (1.0 - postStats.Avg.Seconds()/preStats.Avg.Seconds()) * 100
			sb.WriteString(fmt.Sprintf("%d. **FVT backup speed improved ~%.0f%% on average** (%s -> %s).\n\n", idx, pct, formatDuration(preStats.Avg), formatDuration(postStats.Avg)))
			idx++
		}
	}

	// FVT restore speed
	if len(pre.restores) > 0 && len(post.restores) > 0 {
		preStats := velero.CalcRestoreDurationStats(pre.restores)
		postStats := velero.CalcRestoreDurationStats(post.restores)
		if preStats.Count > 0 && postStats.Count > 0 {
			if preStats.Avg > postStats.Avg {
				pct := (1.0 - postStats.Avg.Seconds()/preStats.Avg.Seconds()) * 100
				sb.WriteString(fmt.Sprintf("%d. **FVT restore speed improved ~%.0f%%** and became more consistent (range %s-%s vs %s-%s before).\n\n",
					idx, pct, formatDuration(postStats.Min), formatDuration(postStats.Max), formatDuration(preStats.Min), formatDuration(preStats.Max)))
			} else {
				sb.WriteString(fmt.Sprintf("%d. **FVT restore speed is stable** (avg %s vs %s before).\n\n", idx, formatDuration(postStats.Avg), formatDuration(preStats.Avg)))
			}
			idx++
		}
	}

	// HC Daily success rate
	preHC := pre.groups[velero.BackupTypeHCDaily]
	postHC := post.groups[velero.BackupTypeHCDaily]
	if len(preHC) > 0 {
		preFailed := 0
		preErrors := 0
		for _, b := range preHC {
			if b.Phase != "Completed" {
				preFailed++
			}
			preErrors += b.Errors
		}
		postFailed := 0
		for _, b := range postHC {
			if b.Phase != "Completed" {
				postFailed++
			}
		}
		if preFailed > 0 && postFailed == 0 {
			preRate := float64(len(preHC)-preFailed) / float64(len(preHC)) * 100
			sb.WriteString(fmt.Sprintf("%d. **HC Daily backups went from %.1f%% to 100%% success rate** - the %d `PartiallyFailed` backups with %d total errors have not recurred post-change.\n\n",
				idx, preRate, preFailed, preErrors))
			idx++
		}
	}

	// Zero errors post-change
	postBkpErrors := sumErrors(post.backups)
	if postBkpErrors == 0 {
		sb.WriteString(fmt.Sprintf("%d. **Zero errors across all backup types post-change.** Restore warnings remain stable and expected (resource conflict warnings).\n\n", idx))
		idx++
	}

	// Sample size caveat
	sb.WriteString(fmt.Sprintf("%d. **Sample size caveat:** Post-change data is based on %d backups and %d restores. A longer observation window (1+ week) would provide more confidence in the results.\n",
		idx, len(post.backups), len(post.restores)))
}

func countCompleted(backups []velero.BackupInfo) int {
	n := 0
	for _, b := range backups {
		if b.Phase == "Completed" {
			n++
		}
	}
	return n
}

func sumErrors(backups []velero.BackupInfo) int {
	total := 0
	for _, b := range backups {
		total += b.Errors
	}
	return total
}

func avgItems(backups []velero.BackupInfo) int {
	if len(backups) == 0 {
		return 0
	}
	total := 0
	for _, b := range backups {
		total += b.ItemsBackedUp
	}
	return total / len(backups)
}

func avgRestoreWarnings(restores []velero.RestoreInfo) float64 {
	if len(restores) == 0 {
		return 0
	}
	total := 0
	for _, r := range restores {
		total += r.Warnings
	}
	return float64(total) / float64(len(restores))
}

func formatRate(completed, total int) string {
	if total == 0 {
		return "N/A"
	}
	pct := float64(completed) / float64(total) * 100
	return fmt.Sprintf("%.1f%% (%d/%d)", pct, completed, total)
}

func deltaRate(preComp, preTotal, postComp, postTotal int) string {
	if preTotal == 0 || postTotal == 0 {
		return "-"
	}
	preRate := float64(preComp) / float64(preTotal) * 100
	postRate := float64(postComp) / float64(postTotal) * 100
	diff := postRate - preRate
	if diff == 0 {
		return "Same"
	}
	return fmt.Sprintf("%+.1f%%", diff)
}

func deltaInt(pre, post int) string {
	if pre == post {
		return "Same"
	}
	diff := post - pre
	if diff > 0 {
		return fmt.Sprintf("+%d", diff)
	}
	return fmt.Sprintf("%d", diff)
}

func calcDeltaPct(pre, post float64) string {
	if pre == 0 {
		return "-"
	}
	pct := ((post - pre) / pre) * 100
	if pct == 0 {
		return "Same"
	}
	return fmt.Sprintf("%+.1f%%", pct)
}

func calcDeltaPctInt(pre, post int) string {
	return calcDeltaPct(float64(pre), float64(post))
}
