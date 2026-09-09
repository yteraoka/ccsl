package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// formatDuration renders a millisecond span compactly: 850ms, 45s, 3m20s,
// 1h04m. Anything an hour or longer drops seconds to stay short.
func formatDuration(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	d := time.Duration(ms) * time.Millisecond
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", ms)
	case d < time.Minute:
		return fmt.Sprintf("%.0fs", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// formatCost keeps small session costs readable instead of rounding them to
// $0.00, and drops to two decimals once the amount is worth noticing.
func formatCost(usd float64) string {
	switch {
	case usd <= 0:
		return "$0.00"
	case usd < 0.01:
		return fmt.Sprintf("$%.4f", usd)
	case usd < 1:
		return fmt.Sprintf("$%.3f", usd)
	default:
		return fmt.Sprintf("$%.2f", usd)
	}
}

// formatTokens abbreviates token counts (12345 -> 12.3k, 1200000 -> 1.2M).
func formatTokens(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 1_000_000:
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1000), ".0") + "k"
	default:
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1_000_000), ".0") + "M"
	}
}

// formatPercent prints whole percents below 10% with one decimal so early
// session movement is still visible.
func formatPercent(p float64) string {
	if p < 10 {
		return strings.TrimSuffix(fmt.Sprintf("%.1f", p), ".0") + "%"
	}
	return fmt.Sprintf("%.0f%%", p)
}

// formatResetIn renders how long until a rate-limit window resets. Empty when
// the timestamp is missing or already in the past.
func formatResetIn(resetsAt int64, now time.Time) string {
	if resetsAt <= 0 {
		return ""
	}
	d := time.Unix(resetsAt, 0).Sub(now)
	if d <= 0 {
		return ""
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes())+1)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd%dh", int(d.Hours())/24, int(d.Hours())%24)
}

// bar draws a fixed-width usage meter. Values above 100% clamp to full.
func bar(pct float64, width int) string {
	if width <= 0 {
		return ""
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct/100*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// shortenPath replaces $HOME with ~ and, when the result is still longer than
// max, keeps the trailing path segments behind an ellipsis.
func shortenPath(p string, maxWidth int) string {
	if p == "" {
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if p == home {
			return "~"
		}
		if strings.HasPrefix(p, home+string(filepath.Separator)) {
			p = "~" + p[len(home):]
		}
	}
	if maxWidth <= 0 || displayWidth(p) <= maxWidth {
		return p
	}
	parts := strings.Split(p, string(filepath.Separator))
	for i := 1; i < len(parts); i++ {
		candidate := "…" + string(filepath.Separator) + strings.Join(parts[i:], string(filepath.Separator))
		if displayWidth(candidate) <= maxWidth {
			return candidate
		}
	}
	// A single segment longer than maxWidth: keep its tail.
	last := parts[len(parts)-1]
	r := []rune(last)
	for len(r) > 0 && displayWidth("…"+string(r)) > maxWidth {
		r = r[1:]
	}
	return "…" + string(r)
}
