package velero

import (
	"testing"
	"time"
)

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZero bool
		wantTime time.Time
	}{
		{"valid RFC3339", "2026-03-05T12:27:46Z", false, time.Date(2026, 3, 5, 12, 27, 46, 0, time.UTC)},
		{"valid with offset", "2026-03-05T12:27:46+00:00", false, time.Date(2026, 3, 5, 12, 27, 46, 0, time.UTC)},
		{"empty string", "", true, time.Time{}},
		{"invalid format", "not-a-date", true, time.Time{}},
		{"partial date", "2026-03-05", true, time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTimestamp(tt.input)
			if tt.wantZero && !got.IsZero() {
				t.Errorf("parseTimestamp(%q) = %v, want zero", tt.input, got)
			}
			if !tt.wantZero && !got.Equal(tt.wantTime) {
				t.Errorf("parseTimestamp(%q) = %v, want %v", tt.input, got, tt.wantTime)
			}
		})
	}
}

func TestGetNestedMap(t *testing.T) {
	obj := map[string]interface{}{
		"status": map[string]interface{}{
			"phase": "Completed",
		},
		"notamap": "string-value",
	}

	m, ok := getNestedMap(obj, "status")
	if !ok || m == nil {
		t.Fatal("expected to find status map")
	}
	if m["phase"] != "Completed" {
		t.Errorf("phase = %v, want Completed", m["phase"])
	}

	_, ok = getNestedMap(obj, "missing")
	if ok {
		t.Error("expected false for missing key")
	}

	_, ok = getNestedMap(obj, "notamap")
	if ok {
		t.Error("expected false for non-map value")
	}
}

func TestGetStringField(t *testing.T) {
	m := map[string]interface{}{
		"name":   "test-backup",
		"count":  42,
		"nested": map[string]interface{}{},
	}

	if got := getStringField(m, "name"); got != "test-backup" {
		t.Errorf("got %q, want %q", got, "test-backup")
	}
	if got := getStringField(m, "missing"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := getStringField(m, "count"); got != "" {
		t.Errorf("got %q for int field, want empty", got)
	}
	if got := getStringField(nil, "key"); got != "" {
		t.Errorf("got %q for nil map, want empty", got)
	}
}

func TestGetIntField(t *testing.T) {
	m := map[string]interface{}{
		"float":  float64(42),
		"int64":  int64(100),
		"int":    200,
		"string": "notanint",
	}

	tests := []struct {
		key      string
		expected int
	}{
		{"float", 42},
		{"int64", 100},
		{"int", 200},
		{"string", 0},
		{"missing", 0},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := getIntField(m, tt.key); got != tt.expected {
				t.Errorf("getIntField(%q) = %d, want %d", tt.key, got, tt.expected)
			}
		})
	}

	if got := getIntField(nil, "key"); got != 0 {
		t.Errorf("got %d for nil map, want 0", got)
	}
}

func TestGetStringSlice(t *testing.T) {
	m := map[string]interface{}{
		"namespaces": []interface{}{"ns1", "ns2", "ns3"},
		"mixed":      []interface{}{"valid", 42, "also-valid"},
		"notslice":   "string",
	}

	got := getStringSlice(m, "namespaces")
	if len(got) != 3 || got[0] != "ns1" || got[1] != "ns2" || got[2] != "ns3" {
		t.Errorf("got %v, want [ns1 ns2 ns3]", got)
	}

	got = getStringSlice(m, "mixed")
	if len(got) != 2 || got[0] != "valid" || got[1] != "also-valid" {
		t.Errorf("got %v, want [valid also-valid]", got)
	}

	got = getStringSlice(m, "notslice")
	if got != nil {
		t.Errorf("got %v for non-slice, want nil", got)
	}

	got = getStringSlice(m, "missing")
	if got != nil {
		t.Errorf("got %v for missing key, want nil", got)
	}

	got = getStringSlice(nil, "key")
	if got != nil {
		t.Errorf("got %v for nil map, want nil", got)
	}
}
