package main

import (
	"strings"
	"unicode"
)

// displayWidth measures a string as a terminal renders it: ANSI/OSC escape
// sequences count as nothing, emoji and CJK take two cells, combining marks
// and variation selectors take none.
func displayWidth(s string) int {
	w, prev := 0, 0
	for _, seg := range splitEscapes(s) {
		if seg.escape {
			continue
		}
		for _, r := range seg.text {
			// U+FE0F asks for emoji presentation, which widens an otherwise
			// narrow base character to two cells.
			if r == 0xFE0F {
				if prev == 1 {
					w++
					prev = 2
				}
				continue
			}
			cw := runeWidth(r)
			w += cw
			prev = cw
		}
	}
	return w
}

func runeWidth(r rune) int {
	switch {
	case r == 0x200D, r == 0xFE0E, r == 0xFE0F: // ZWJ, variation selectors (FE0F handled by caller)
		return 0
	case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Me, r), unicode.Is(unicode.Cf, r):
		return 0
	case r < 0x1100:
		return 1
	case isWide(r):
		return 2
	default:
		return 1
	}
}

// isWide reports the East Asian Wide / emoji-presentation ranges that occupy
// two terminal cells. It covers the ranges this status line can emit rather
// than the whole Unicode table.
func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r >= 0x2E80 && r <= 0x303E, // CJK radicals, Kangxi
		r >= 0x3041 && r <= 0x33FF, // Kana, CJK compatibility
		r >= 0x3400 && r <= 0x4DBF, // CJK ext A
		r >= 0x4E00 && r <= 0x9FFF, // CJK unified
		r >= 0xA000 && r <= 0xA4CF, // Yi
		r >= 0xAC00 && r <= 0xD7A3, // Hangul syllables
		r >= 0xF900 && r <= 0xFAFF, // CJK compatibility ideographs
		r >= 0xFE10 && r <= 0xFE19,
		r >= 0xFE30 && r <= 0xFE6F,
		r >= 0xFF00 && r <= 0xFF60, // Fullwidth forms
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x231A && r <= 0x231B, // emoji-presentation singles and ranges
		r >= 0x23E9 && r <= 0x23EC,
		r == 0x23F0, r == 0x23F3,
		r >= 0x25FD && r <= 0x25FE,
		r >= 0x2614 && r <= 0x2615,
		r >= 0x2648 && r <= 0x2653,
		r == 0x267F, r == 0x2693, r == 0x26A1,
		r >= 0x26AA && r <= 0x26AB,
		r >= 0x26BD && r <= 0x26BE,
		r >= 0x26C4 && r <= 0x26C5,
		r == 0x26CE, r == 0x26D4, r == 0x26EA,
		r >= 0x26F2 && r <= 0x26F3,
		r == 0x26F5, r == 0x26FA, r == 0x26FD,
		r == 0x2705,
		r >= 0x270A && r <= 0x270B,
		r == 0x2728, r == 0x274C, r == 0x274E,
		r >= 0x2753 && r <= 0x2755,
		r == 0x2757,
		r >= 0x2795 && r <= 0x2797,
		r == 0x27B0, r == 0x27BF,
		r >= 0x2B1B && r <= 0x2B1C,
		r == 0x2B50, r == 0x2B55,
		r >= 0x1F300 && r <= 0x1F64F, // emoji
		r >= 0x1F680 && r <= 0x1F6FF,
		r >= 0x1F900 && r <= 0x1F9FF,
		r >= 0x1FA70 && r <= 0x1FAFF,
		r >= 0x20000 && r <= 0x3FFFD:
		return true
	}
	return false
}

type segment struct {
	text   string
	escape bool
}

// splitEscapes separates ANSI CSI sequences and OSC 8 hyperlink wrappers from
// printable text so widths and truncation ignore them.
func splitEscapes(s string) []segment {
	var segs []segment
	for len(s) > 0 {
		i := strings.IndexByte(s, 0x1b)
		if i < 0 {
			segs = append(segs, segment{text: s})
			break
		}
		if i > 0 {
			segs = append(segs, segment{text: s[:i]})
		}
		rest := s[i:]
		n := escapeLen(rest)
		if n == 0 { // lone ESC: treat as text so we never loop forever
			segs = append(segs, segment{text: rest[:1]})
			n = 1
		} else {
			segs = append(segs, segment{text: rest[:n], escape: true})
		}
		s = rest[n:]
	}
	return segs
}

// escapeLen returns the byte length of the escape sequence starting at s[0],
// or 0 when s does not start with one we recognize.
func escapeLen(s string) int {
	if len(s) < 2 || s[0] != 0x1b {
		return 0
	}
	switch s[1] {
	case '[': // CSI ... final byte in @-~
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
		return len(s)
	case ']': // OSC ... terminated by BEL or ST (ESC \)
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return len(s)
	}
	return 0
}

// truncateToWidth cuts s to at most max terminal cells, appending an ellipsis.
// Escape sequences are preserved (they cost no width) and the result is closed
// with a reset — plus an OSC 8 terminator when the cut fell inside a link — so
// truncation never leaks styling into the rest of the terminal.
func truncateToWidth(s string, max int) string {
	if max <= 0 || displayWidth(s) <= max {
		return s
	}
	var b strings.Builder
	w, prev, truncated := 0, 0, false

	for _, seg := range splitEscapes(s) {
		if seg.escape {
			b.WriteString(seg.text)
			continue
		}
		for _, r := range seg.text {
			cw := runeWidth(r)
			if r == 0xFE0F && prev == 1 {
				cw = 1
			}
			if w+cw > max-1 {
				truncated = true
				break
			}
			b.WriteRune(r)
			w += cw
			prev = cw
		}
		if truncated {
			break
		}
	}
	b.WriteString("…")
	if strings.Contains(s, "\033]8;;") {
		b.WriteString("\033]8;;\033\\")
	}
	if strings.Contains(s, "\033[") {
		b.WriteString(ansiReset)
	}
	return b.String()
}
