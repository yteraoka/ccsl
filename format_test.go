package main

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{-1, "0ms"},
		{0, "0ms"},
		{850, "850ms"},
		{45000, "45s"},
		{200000, "3m20s"},
		{4500000, "1h15m"},
		{90061000, "25h01m"},
	}
	for _, tt := range tests {
		if got := formatDuration(tt.ms); got != tt.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		usd  float64
		want string
	}{
		{0, "$0.00"},
		{-1, "$0.00"},
		{0.0001234, "$0.0001"},
		{0.1234, "$0.123"},
		{1.2345, "$1.23"},
		{123.456, "$123.46"},
	}
	for _, tt := range tests {
		if got := formatCost(tt.usd); got != tt.want {
			t.Errorf("formatCost(%v) = %q, want %q", tt.usd, got, tt.want)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1k"},
		{12345, "12.3k"},
		{200000, "200k"},
		{1200000, "1.2M"},
	}
	for _, tt := range tests {
		if got := formatTokens(tt.n); got != tt.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		p    float64
		want string
	}{
		{0, "0%"},
		{4.25, "4.2%"},
		{9.9, "9.9%"},
		{42.85, "43%"},
		{100, "100%"},
	}
	for _, tt := range tests {
		if got := formatPercent(tt.p); got != tt.want {
			t.Errorf("formatPercent(%v) = %q, want %q", tt.p, got, tt.want)
		}
	}
}

func TestFormatResetIn(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	tests := []struct {
		resets int64
		want   string
	}{
		{0, ""},
		{now.Unix() - 60, ""}, // already reset
		{now.Add(90 * time.Second).Unix(), "2m"},
		{now.Add(2*time.Hour + 10*time.Minute).Unix(), "2h10m"},
		{now.Add(50 * time.Hour).Unix(), "2d2h"},
	}
	for _, tt := range tests {
		if got := formatResetIn(tt.resets, now); got != tt.want {
			t.Errorf("formatResetIn(%d) = %q, want %q", tt.resets, got, tt.want)
		}
	}
}

func TestBar(t *testing.T) {
	tests := []struct {
		pct   float64
		width int
		want  string
	}{
		{0, 10, "░░░░░░░░░░"},
		{50, 10, "█████░░░░░"},
		{100, 10, "██████████"},
		{150, 10, "██████████"}, // clamped
		{-5, 10, "░░░░░░░░░░"},  // clamped
		{50, 0, ""},
	}
	for _, tt := range tests {
		if got := bar(tt.pct, tt.width); got != tt.want {
			t.Errorf("bar(%v, %d) = %q, want %q", tt.pct, tt.width, got, tt.want)
		}
	}
}

func TestShortenPath(t *testing.T) {
	t.Setenv("HOME", "/home/u")
	tests := []struct {
		p    string
		max  int
		want string
	}{
		{"", 0, ""},
		{"/home/u", 0, "~"},
		{"/home/u/src/app", 0, "~/src/app"},
		{"/home/u/src/app", 20, "~/src/app"},
		{"/home/u/very/deep/nested/project", 12, "…/project"},
		{"/averyveryverylongsinglesegment", 10, "…lesegment"},
	}
	for _, tt := range tests {
		if got := shortenPath(tt.p, tt.max); got != tt.want {
			t.Errorf("shortenPath(%q, %d) = %q, want %q", tt.p, tt.max, got, tt.want)
		}
	}
}
