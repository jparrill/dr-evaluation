package velero

import (
	"strings"
	"time"
)

type BackupType string

const (
	BackupTypeFVT       BackupType = "FVT"
	BackupTypeDailyFull BackupType = "DailyFull"
	BackupTypeHCDaily   BackupType = "HCDaily"
	BackupTypeOther     BackupType = "Other"
)

type BackupInfo struct {
	Name                string
	Type                BackupType
	CreationTimestamp   time.Time
	StartTimestamp      time.Time
	CompletionTimestamp time.Time
	Duration            time.Duration
	Phase               string
	ItemsBackedUp       int
	TotalItems          int
	Warnings            int
	Errors              int
	TTL                 string
	IncludedNamespaces  []string
	StorageLocation     string
}

type RestoreInfo struct {
	Name                string
	BackupName          string
	CreationTimestamp   time.Time
	StartTimestamp      time.Time
	CompletionTimestamp time.Time
	Duration            time.Duration
	Phase               string
	ItemsRestored       int
	TotalItems          int
	Warnings            int
	Errors              int
}

func ClassifyBackup(name string) BackupType {
	if strings.HasSuffix(name, "-bkp-fvt") {
		return BackupTypeFVT
	}
	if strings.HasPrefix(name, "daily-full-backup-") {
		return BackupTypeDailyFull
	}
	if strings.Contains(name, "-daily-") {
		return BackupTypeHCDaily
	}
	return BackupTypeOther
}

func FilterBackupsByTime(backups []BackupInfo, start, end time.Time) []BackupInfo {
	var filtered []BackupInfo
	for _, b := range backups {
		if !b.CreationTimestamp.Before(start) && !b.CreationTimestamp.After(end) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

func FilterRestoresByTime(restores []RestoreInfo, start, end time.Time) []RestoreInfo {
	var filtered []RestoreInfo
	for _, r := range restores {
		if !r.CreationTimestamp.Before(start) && !r.CreationTimestamp.After(end) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func GroupBackupsByType(backups []BackupInfo) map[BackupType][]BackupInfo {
	groups := make(map[BackupType][]BackupInfo)
	for _, b := range backups {
		groups[b.Type] = append(groups[b.Type], b)
	}
	return groups
}

func BuildRestoreMap(restores []RestoreInfo) map[string][]RestoreInfo {
	m := make(map[string][]RestoreInfo)
	for _, r := range restores {
		if r.BackupName != "" {
			m[r.BackupName] = append(m[r.BackupName], r)
		}
	}
	return m
}

func SampleLast(backups []BackupInfo, n int) []BackupInfo {
	if len(backups) <= n {
		return backups
	}
	return backups[len(backups)-n:]
}

func SampleLastRestores(restores []RestoreInfo, n int) []RestoreInfo {
	if len(restores) <= n {
		return restores
	}
	return restores[len(restores)-n:]
}

type DurationStats struct {
	Min   time.Duration
	Max   time.Duration
	Avg   time.Duration
	Count int
}

func CalcBackupDurationStats(backups []BackupInfo) DurationStats {
	var stats DurationStats
	var total time.Duration
	for _, b := range backups {
		if b.Duration <= 0 {
			continue
		}
		stats.Count++
		total += b.Duration
		if stats.Count == 1 || b.Duration < stats.Min {
			stats.Min = b.Duration
		}
		if b.Duration > stats.Max {
			stats.Max = b.Duration
		}
	}
	if stats.Count > 0 {
		stats.Avg = total / time.Duration(stats.Count)
	}
	return stats
}

func CalcRestoreDurationStats(restores []RestoreInfo) DurationStats {
	var stats DurationStats
	var total time.Duration
	for _, r := range restores {
		if r.Duration <= 0 {
			continue
		}
		stats.Count++
		total += r.Duration
		if stats.Count == 1 || r.Duration < stats.Min {
			stats.Min = r.Duration
		}
		if r.Duration > stats.Max {
			stats.Max = r.Duration
		}
	}
	if stats.Count > 0 {
		stats.Avg = total / time.Duration(stats.Count)
	}
	return stats
}
