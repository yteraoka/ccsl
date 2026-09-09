package main

import "fmt"

// ANSI SGR codes used by the status line. Bright variants read well on both
// light and dark terminal themes.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[91m"
	ansiGreen  = "\033[92m"
	ansiYellow = "\033[93m"
	ansiBlue   = "\033[94m"
	ansiPurple = "\033[95m"
	ansiCyan   = "\033[96m"
	ansiGray   = "\033[90m"
)

type painter struct {
	enabled bool
	links   bool
}

func (p painter) paint(color, s string) string {
	if !p.enabled || s == "" || color == "" {
		return s
	}
	return color + s + ansiReset
}

// usageColor grades a 0-100 percentage: green while there is room, yellow as
// it tightens, red once it is nearly spent.
func usageColor(pct float64) string {
	switch {
	case pct >= 90:
		return ansiRed
	case pct >= 70:
		return ansiYellow
	default:
		return ansiGreen
	}
}

// link wraps text in an OSC 8 hyperlink so terminals that support it make the
// label clickable, and leaves the label untouched everywhere else.
func (p painter) link(url, text string) string {
	if !p.links || url == "" {
		return text
	}
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}
