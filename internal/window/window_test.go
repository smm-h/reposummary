package window

import (
	"strings"
	"testing"
)

func TestParseWindowForms(t *testing.T) {
	tests := []struct {
		spec      string
		wantMode  string
		wantLabel string
	}{
		{"today", "daterange", "Today ("},
		{"yesterday", "daterange", "Yesterday ("},
		{"week", "daterange", "Last 7 days"},
		{"7d", "daterange", "Last 7 days"},
		{"month", "daterange", "Last 30 days"},
		{"30d", "daterange", "Last 30 days"},
		{"all", "range", "Full history"},
		{"start", "range", "Full history"},
		{"start..now", "range", "Full history"},
	}
	for _, tc := range tests {
		got, err := ParseWindow(tc.spec)
		if err != nil {
			t.Fatalf("ParseWindow(%q) error: %v", tc.spec, err)
		}
		if got.Mode != tc.wantMode {
			t.Errorf("ParseWindow(%q).Mode = %q, want %q", tc.spec, got.Mode, tc.wantMode)
		}
		if !strings.HasPrefix(got.Label, tc.wantLabel) {
			t.Errorf("ParseWindow(%q).Label = %q, want prefix %q", tc.spec, got.Label, tc.wantLabel)
		}
	}
}

func TestParseWindowRevRange(t *testing.T) {
	got, err := ParseWindow("v0.1.0..HEAD")
	if err != nil {
		t.Fatalf("ParseWindow error: %v", err)
	}
	if got.Mode != "range" {
		t.Errorf("Mode = %q, want range", got.Mode)
	}
	if got.RevRange != "v0.1.0..HEAD" {
		t.Errorf("RevRange = %q, want v0.1.0..HEAD", got.RevRange)
	}
	if got.Label != "v0.1.0..HEAD" {
		t.Errorf("Label = %q, want v0.1.0..HEAD", got.Label)
	}
}

func TestParseWindowDateAdd(t *testing.T) {
	got, err := ParseWindow("2026-06-11+1month")
	if err != nil {
		t.Fatalf("ParseWindow error: %v", err)
	}
	if got.Mode != "daterange" {
		t.Errorf("Mode = %q, want daterange", got.Mode)
	}
	if got.Since != "2026-06-11" {
		t.Errorf("Since = %q, want 2026-06-11", got.Since)
	}
	if got.Until != "2026-07-11" {
		t.Errorf("Until = %q, want 2026-07-11", got.Until)
	}
	if got.Label != "2026-06-11 -> 2026-07-11" {
		t.Errorf("Label = %q, want '2026-06-11 -> 2026-07-11'", got.Label)
	}
}

func TestParseWindowDateAddUnits(t *testing.T) {
	tests := []struct {
		spec      string
		wantUntil string
	}{
		{"2026-01-01+5days", "2026-01-06"},
		{"2026-01-01+2weeks", "2026-01-15"},
		{"2026-01-01+3y", "2029-01-01"},
		{"2026-01-01+10d", "2026-01-11"},
	}
	for _, tc := range tests {
		got, err := ParseWindow(tc.spec)
		if err != nil {
			t.Fatalf("ParseWindow(%q) error: %v", tc.spec, err)
		}
		if got.Until != tc.wantUntil {
			t.Errorf("ParseWindow(%q).Until = %q, want %q", tc.spec, got.Until, tc.wantUntil)
		}
	}
}

func TestParseWindowBogus(t *testing.T) {
	_, err := ParseWindow("nonsense-window")
	if err == nil {
		t.Fatal("ParseWindow(bogus) should error")
	}
	if !strings.Contains(err.Error(), "supported forms") {
		t.Errorf("error %q should list supported forms", err.Error())
	}
}
