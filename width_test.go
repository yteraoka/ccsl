package main

import (
	"strings"
	"testing"
)

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"\033[92mabc\033[0m", 3},
		{"🤖", 2},
		{"⏱️", 2}, // base char plus a zero-width variation selector
		{"日本語", 6},
		{"✅", 2},
		{"│", 1},
		{"\033]8;;https://example.com\033\\link\033]8;;\033\\", 4},
		{"\033[1;31m日\033[0m x", 4},
	}
	for _, tt := range tests {
		if got := displayWidth(tt.s); got != tt.want {
			t.Errorf("displayWidth(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}

func TestDisplayWidthLoneEscape(t *testing.T) {
	// A bare ESC must not send splitEscapes into an infinite loop.
	if got := displayWidth("\033"); got != 1 {
		t.Errorf("displayWidth(ESC) = %d, want 1", got)
	}
}

func TestTruncateToWidth(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"abcdef", 0, "abcdef"},  // no limit
		{"abcdef", 10, "abcdef"}, // already fits
		{"abcdef", 4, "abc…"},
		{"日本語です", 5, "日本…"},
	}
	for _, tt := range tests {
		if got := truncateToWidth(tt.s, tt.max); got != tt.want {
			t.Errorf("truncateToWidth(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
		}
	}
}

func TestTruncateToWidthClosesEscapes(t *testing.T) {
	s := "\033[92mhello world\033[0m"
	got := truncateToWidth(s, 6)
	if displayWidth(got) > 6 {
		t.Errorf("width %d > 6: %q", displayWidth(got), got)
	}
	if !strings.HasSuffix(got, ansiReset) {
		t.Errorf("truncated colored text must end with a reset: %q", got)
	}

	link := "\033]8;;https://example.com\033\\a very long link label\033]8;;\033\\"
	got = truncateToWidth(link, 8)
	if !strings.HasSuffix(got, "\033]8;;\033\\") {
		t.Errorf("truncated hyperlink must be closed: %q", got)
	}
}
