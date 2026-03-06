package velero

import (
	"testing"
	"time"
)

func TestClassifyBackup(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected BackupType
	}{
		{"FVT suffix", "2ori8kv5vprd4v4vasabfb7rguhjoh36-bkp-fvt", BackupTypeFVT},
		{"daily full prefix", "daily-full-backup-20260306000027", BackupTypeDailyFull},
		{"HC daily with daily infix", "2oro9afenthes8rhtvftp7qil9e6tt6s-daily-20260306050028", BackupTypeHCDaily},
		{"other backup", "weekly-full-backup-20260302010058", BackupTypeOther},
		{"empty name", "", BackupTypeOther},
		{"FVT must be suffix", "bkp-fvt-something", BackupTypeOther},
		{"daily-full must be prefix", "something-daily-full-backup-123", BackupTypeHCDaily}, // contains -daily-
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyBackup(tt.input)
			if got != tt.expected {
				t.Errorf("ClassifyBackup(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFilterBackupsByTime(t *testing.T) {
	base := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	backups := []BackupInfo{
		{Name: "b1", CreationTimestamp: base.Add(-48 * time.Hour)},
		{Name: "b2", CreationTimestamp: base.Add(-24 * time.Hour)},
		{Name: "b3", CreationTimestamp: base},
		{Name: "b4", CreationTimestamp: base.Add(12 * time.Hour)},
		{Name: "b5", CreationTimestamp: base.Add(48 * time.Hour)},
	}

	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected []string
	}{
		{
			"full range",
			base.Add(-48 * time.Hour),
			base.Add(48 * time.Hour),
			[]string{"b1", "b2", "b3", "b4", "b5"},
		},
		{
			"narrow range",
			base,
			base.Add(12 * time.Hour),
			[]string{"b3", "b4"},
		},
		{
			"no matches",
			base.Add(100 * time.Hour),
			base.Add(200 * time.Hour),
			nil,
		},
		{
			"exact boundary",
			base,
			base,
			[]string{"b3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterBackupsByTime(backups, tt.start, tt.end)
			if len(got) != len(tt.expected) {
				t.Fatalf("got %d backups, want %d", len(got), len(tt.expected))
			}
			for i, b := range got {
				if b.Name != tt.expected[i] {
					t.Errorf("got[%d].Name = %q, want %q", i, b.Name, tt.expected[i])
				}
			}
		})
	}
}

func TestFilterRestoresByTime(t *testing.T) {
	base := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	restores := []RestoreInfo{
		{Name: "r1", CreationTimestamp: base.Add(-1 * time.Hour)},
		{Name: "r2", CreationTimestamp: base},
		{Name: "r3", CreationTimestamp: base.Add(1 * time.Hour)},
	}

	got := FilterRestoresByTime(restores, base, base.Add(1*time.Hour))
	if len(got) != 2 {
		t.Fatalf("got %d restores, want 2", len(got))
	}
	if got[0].Name != "r2" || got[1].Name != "r3" {
		t.Errorf("unexpected restores: %v", got)
	}
}

func TestFilterBackupsByTimeEmpty(t *testing.T) {
	got := FilterBackupsByTime(nil, time.Time{}, time.Now())
	if got != nil {
		t.Errorf("expected nil for nil input, got %v", got)
	}
}

func TestGroupBackupsByType(t *testing.T) {
	backups := []BackupInfo{
		{Name: "a", Type: BackupTypeFVT},
		{Name: "b", Type: BackupTypeFVT},
		{Name: "c", Type: BackupTypeDailyFull},
		{Name: "d", Type: BackupTypeHCDaily},
		{Name: "e", Type: BackupTypeOther},
		{Name: "f", Type: BackupTypeOther},
	}

	groups := GroupBackupsByType(backups)

	if len(groups[BackupTypeFVT]) != 2 {
		t.Errorf("FVT count = %d, want 2", len(groups[BackupTypeFVT]))
	}
	if len(groups[BackupTypeDailyFull]) != 1 {
		t.Errorf("DailyFull count = %d, want 1", len(groups[BackupTypeDailyFull]))
	}
	if len(groups[BackupTypeHCDaily]) != 1 {
		t.Errorf("HCDaily count = %d, want 1", len(groups[BackupTypeHCDaily]))
	}
	if len(groups[BackupTypeOther]) != 2 {
		t.Errorf("Other count = %d, want 2", len(groups[BackupTypeOther]))
	}
}

func TestGroupBackupsByTypeEmpty(t *testing.T) {
	groups := GroupBackupsByType(nil)
	if len(groups) != 0 {
		t.Errorf("expected empty map, got %d entries", len(groups))
	}
}

func TestBuildRestoreMap(t *testing.T) {
	restores := []RestoreInfo{
		{Name: "r1", BackupName: "b1"},
		{Name: "r2", BackupName: "b1"},
		{Name: "r3", BackupName: "b2"},
		{Name: "r4", BackupName: ""},
	}

	m := BuildRestoreMap(restores)
	if len(m["b1"]) != 2 {
		t.Errorf("b1 restores = %d, want 2", len(m["b1"]))
	}
	if len(m["b2"]) != 1 {
		t.Errorf("b2 restores = %d, want 1", len(m["b2"]))
	}
	if _, ok := m[""]; ok {
		t.Error("empty backup name should not be in map")
	}
}

func TestSampleLast(t *testing.T) {
	backups := []BackupInfo{
		{Name: "b1"}, {Name: "b2"}, {Name: "b3"}, {Name: "b4"}, {Name: "b5"},
	}

	tests := []struct {
		name     string
		n        int
		expected []string
	}{
		{"sample 3 of 5", 3, []string{"b3", "b4", "b5"}},
		{"sample 5 of 5", 5, []string{"b1", "b2", "b3", "b4", "b5"}},
		{"sample 10 of 5", 10, []string{"b1", "b2", "b3", "b4", "b5"}},
		{"sample 1 of 5", 1, []string{"b5"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SampleLast(backups, tt.n)
			if len(got) != len(tt.expected) {
				t.Fatalf("got %d, want %d", len(got), len(tt.expected))
			}
			for i, b := range got {
				if b.Name != tt.expected[i] {
					t.Errorf("got[%d] = %q, want %q", i, b.Name, tt.expected[i])
				}
			}
		})
	}
}

func TestSampleLastRestores(t *testing.T) {
	restores := []RestoreInfo{{Name: "r1"}, {Name: "r2"}, {Name: "r3"}}

	got := SampleLastRestores(restores, 2)
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
	if got[0].Name != "r2" || got[1].Name != "r3" {
		t.Errorf("unexpected result: %v", got)
	}

	got = SampleLastRestores(restores, 10)
	if len(got) != 3 {
		t.Fatalf("got %d, want 3", len(got))
	}
}

func TestCalcBackupDurationStats(t *testing.T) {
	backups := []BackupInfo{
		{Duration: 100 * time.Second},
		{Duration: 200 * time.Second},
		{Duration: 300 * time.Second},
	}

	stats := CalcBackupDurationStats(backups)

	if stats.Count != 3 {
		t.Errorf("Count = %d, want 3", stats.Count)
	}
	if stats.Min != 100*time.Second {
		t.Errorf("Min = %v, want 100s", stats.Min)
	}
	if stats.Max != 300*time.Second {
		t.Errorf("Max = %v, want 300s", stats.Max)
	}
	if stats.Avg != 200*time.Second {
		t.Errorf("Avg = %v, want 200s", stats.Avg)
	}
}

func TestCalcBackupDurationStatsSkipsZero(t *testing.T) {
	backups := []BackupInfo{
		{Duration: 0},
		{Duration: 60 * time.Second},
		{Duration: -1 * time.Second},
		{Duration: 120 * time.Second},
	}

	stats := CalcBackupDurationStats(backups)
	if stats.Count != 2 {
		t.Errorf("Count = %d, want 2", stats.Count)
	}
	if stats.Min != 60*time.Second {
		t.Errorf("Min = %v, want 60s", stats.Min)
	}
	if stats.Avg != 90*time.Second {
		t.Errorf("Avg = %v, want 90s", stats.Avg)
	}
}

func TestCalcBackupDurationStatsEmpty(t *testing.T) {
	stats := CalcBackupDurationStats(nil)
	if stats.Count != 0 || stats.Min != 0 || stats.Max != 0 || stats.Avg != 0 {
		t.Errorf("expected zero stats for nil input, got %+v", stats)
	}
}

func TestCalcRestoreDurationStats(t *testing.T) {
	restores := []RestoreInfo{
		{Duration: 24 * time.Second},
		{Duration: 26 * time.Second},
		{Duration: 28 * time.Second},
	}

	stats := CalcRestoreDurationStats(restores)

	if stats.Count != 3 {
		t.Errorf("Count = %d, want 3", stats.Count)
	}
	if stats.Min != 24*time.Second {
		t.Errorf("Min = %v, want 24s", stats.Min)
	}
	if stats.Max != 28*time.Second {
		t.Errorf("Max = %v, want 28s", stats.Max)
	}
	if stats.Avg != 26*time.Second {
		t.Errorf("Avg = %v, want 26s", stats.Avg)
	}
}

func TestCalcRestoreDurationStatsEmpty(t *testing.T) {
	stats := CalcRestoreDurationStats(nil)
	if stats.Count != 0 {
		t.Errorf("expected zero count for nil input, got %d", stats.Count)
	}
}
