package main

import (
	"fmt"
	"strings"
	"time"
)

// Options controls how a status line is rendered. Zero values are not valid
// defaults; use defaultOptions.
type Options struct {
	Color      bool
	Emoji      bool
	Links      bool
	SingleLine bool
	GitDirty   bool
	BarWidth   int
	Columns    int
}

// minDirWidth is the narrowest elided path still worth showing.
const minDirWidth = 12

func defaultOptions() Options {
	return Options{
		Color:    true,
		Emoji:    true,
		Links:    true,
		GitDirty: true,
		BarWidth: 10,
	}
}

type renderer struct {
	in   *Input
	opt  Options
	p    painter
	git  gitInfo
	now  time.Time
	sepC string
}

func newRenderer(in *Input, opt Options, git gitInfo, now time.Time) *renderer {
	r := &renderer{
		in:  in,
		opt: opt,
		p:   painter{enabled: opt.Color, links: opt.Links},
		git: git,
		now: now,
	}
	r.sepC = r.p.paint(ansiGray, " │ ")
	return r
}

// icon returns the emoji for a segment, or a short text label when emoji are
// disabled, so both modes stay aligned and readable.
func (r *renderer) icon(emoji, label string) string {
	if r.opt.Emoji {
		return emoji + " "
	}
	if label == "" {
		return ""
	}
	return label + " "
}

// Render produces the full status line: three rows joined by newlines, or a
// single row when Options.SingleLine is set.
func Render(in *Input, opt Options, git gitInfo, now time.Time) string {
	r := newRenderer(in, opt, git, now)

	lines := []string{r.firstLine(), r.secondLine(), r.thirdLine()}

	if opt.SingleLine {
		return truncateToWidth(joinNonEmpty(r.sepC, lines...), opt.Columns)
	}
	for i, l := range lines {
		lines[i] = truncateToWidth(l, opt.Columns)
	}
	return joinNonEmpty("\n", lines...)
}

// firstLine carries the "where am I" context: model, directory, branch,
// worktree and pull request.
func (r *renderer) firstLine() string {
	var segs []string
	dirIdx := -1

	if name := r.in.Model.DisplayName; name != "" {
		segs = append(segs, r.icon("🤖", "model")+r.p.paint(ansiBold+ansiCyan, name))
	}
	if dir := r.dirSegment(0); dir != "" {
		dirIdx = len(segs)
		segs = append(segs, dir)
	}
	if b := r.branchSegment(); b != "" {
		segs = append(segs, b)
	}
	if w := r.worktreeSegment(); w != "" {
		segs = append(segs, w)
	}
	if pr := r.prSegment(); pr != "" {
		segs = append(segs, pr)
	}

	line := strings.Join(segs, r.sepC)
	if r.opt.Columns <= 0 || dirIdx < 0 || displayWidth(line) <= r.opt.Columns {
		return line
	}
	// Too wide: re-render the directory with whatever budget the other
	// segments leave it, keeping its trailing components. Below minDirWidth
	// the path stops being recognizable, so the caller trims the line instead.
	budget := r.opt.Columns - (displayWidth(line) - displayWidth(r.dirText(0)))
	if budget < minDirWidth {
		budget = minDirWidth
	}
	segs[dirIdx] = r.dirSegment(budget)
	return strings.Join(segs, r.sepC)
}

// secondLine carries the "what is it costing" context: context window, prompt
// cache, rate limits and money.
func (r *renderer) secondLine() string {
	var segs []string

	segs = append(segs, r.contextSegment())
	if c := r.cacheSegment(); c != "" {
		segs = append(segs, c)
	}
	if rl := r.rateLimitSegments(); len(rl) > 0 {
		segs = append(segs, rl...)
	}
	segs = append(segs, r.icon("💰", "cost")+r.p.paint(ansiYellow, formatCost(r.in.Cost.TotalCostUSD)))
	return joinNonEmpty(r.sepC, segs...)
}

// thirdLine carries the session id and the elapsed times. They sit apart from
// the second row to keep it from overflowing, with the id leading so the full
// UUID is the part that survives if the row is ever trimmed.
func (r *renderer) thirdLine() string {
	var segs []string
	if r.in.SessionID != "" {
		segs = append(segs, r.icon("🆔", "id")+r.p.paint(ansiGray, r.in.SessionID))
	}
	segs = append(segs,
		r.icon("⏱️", "session")+r.p.paint(ansiBlue, formatDuration(r.in.Cost.TotalDurationMS)),
		r.icon("⚡", "api")+r.p.paint(ansiPurple, formatDuration(r.in.Cost.TotalAPIDurationMS)),
	)
	return joinNonEmpty(r.sepC, segs...)
}

func (r *renderer) dirText(maxWidth int) string {
	return shortenPath(r.in.dir(), maxWidth)
}

func (r *renderer) dirSegment(maxWidth int) string {
	dir := r.dirText(maxWidth)
	if dir == "" {
		return ""
	}
	return r.icon("📁", "dir") + r.p.paint(ansiBlue, dir)
}

// branchSegment shows the checked-out branch plus a dirty marker and the
// ahead/behind counts against its upstream.
func (r *renderer) branchSegment() string {
	if r.git.Branch == "" {
		return ""
	}
	s := r.p.paint(ansiGreen, r.git.Branch)
	if r.git.Dirty {
		s += r.p.paint(ansiYellow, "✱")
	}
	if r.git.Ahead > 0 {
		s += r.p.paint(ansiCyan, fmt.Sprintf(" ↑%d", r.git.Ahead))
	}
	if r.git.Behind > 0 {
		s += r.p.paint(ansiCyan, fmt.Sprintf(" ↓%d", r.git.Behind))
	}
	return r.icon("🌿", "branch") + s
}

// worktreeSegment reports either a Claude Code worktree session or a plain
// linked git worktree, whichever the payload describes.
func (r *renderer) worktreeSegment() string {
	if wt := r.in.Worktree; wt != nil && wt.Name != "" {
		s := r.p.paint(ansiPurple, wt.Name)
		if wt.OriginalBranch != "" {
			s += r.p.paint(ansiGray, " ← "+wt.OriginalBranch)
		}
		return r.icon("🌳", "worktree") + s
	}
	if name := r.in.Workspace.GitWorktree; name != "" {
		return r.icon("🌳", "worktree") + r.p.paint(ansiPurple, name)
	}
	return ""
}

// prSegment renders the open pull request (or GitLab merge request) as a
// clickable link with its review state.
func (r *renderer) prSegment() string {
	pr := r.in.PR
	if pr == nil || pr.Number == 0 {
		return ""
	}
	label := fmt.Sprintf("PR #%d", pr.Number)
	if pr.Kind == "mr" {
		label = fmt.Sprintf("MR !%d", pr.Number)
	}
	color := ansiCyan
	switch pr.ReviewState {
	case "approved":
		color = ansiGreen
	case "changes_requested":
		color = ansiRed
	case "draft":
		color = ansiGray
	}
	s := r.p.link(pr.URL, r.p.paint(color, label))
	if mark := reviewStateMark(pr.ReviewState); mark != "" && r.opt.Emoji {
		s += " " + mark
	} else if mark := pr.ReviewState; mark != "" && !r.opt.Emoji {
		s += r.p.paint(ansiGray, " ("+mark+")")
	}
	return r.icon("🔗", "pr") + s
}

func reviewStateMark(state string) string {
	switch state {
	case "approved":
		return "✅"
	case "changes_requested":
		return "❌"
	case "pending":
		return "👀"
	case "draft":
		return "📝"
	}
	return ""
}

// contextSegment shows how much of the context window the session has burned,
// as a meter, a percentage and the raw token counts.
func (r *renderer) contextSegment() string {
	cw := r.in.ContextWindow
	used := cw.TotalInputTokens + cw.TotalOutputTokens

	pct := 0.0
	known := false
	if cw.UsedPercentage != nil {
		pct, known = *cw.UsedPercentage, true
	} else if cw.ContextWindowSize > 0 {
		pct, known = float64(used)/float64(cw.ContextWindowSize)*100, true
	}
	if !known {
		return r.icon("🧠", "ctx") + r.p.paint(ansiGray, "—")
	}

	color := usageColor(pct)
	s := r.p.paint(color, bar(pct, r.opt.BarWidth)) + " " + r.p.paint(color, formatPercent(pct))
	if cw.ContextWindowSize > 0 {
		s += r.p.paint(ansiGray, fmt.Sprintf(" (%s/%s)", formatTokens(used), formatTokens(cw.ContextWindowSize)))
	}
	if r.in.Exceeds200kTokens {
		s += " " + r.p.paint(ansiRed, "⚠")
	}
	return r.icon("🧠", "ctx") + s
}

// cacheSegment reports the prompt cache: how much of the session's input came
// from cache, how long the warm prefix has left, and whether any request had to
// re-process content the cache already held.
func (r *renderer) cacheSegment() string {
	pc := r.in.PromptCache
	if pc == nil {
		return ""
	}
	icon := r.icon("💾", "cache")
	if !pc.CachingObserved {
		// Caching is off, or the provider does not report cache tokens.
		return icon + r.p.paint(ansiGray, "off")
	}

	var parts []string
	if pc.HitRatio != nil {
		hit := *pc.HitRatio * 100
		parts = append(parts, r.p.paint(cacheColor(hit), formatPercent(hit)))
	}
	if pc.Warm {
		// "12m/1h" reads as 12 minutes left of a one-hour cache lifetime.
		life := pc.TTL
		if pc.ExpiresAt != nil {
			if left := formatResetIn(*pc.ExpiresAt, r.now); left != "" {
				life = strings.TrimSuffix(left+"/"+pc.TTL, "/")
			}
		}
		if life != "" {
			parts = append(parts, r.p.paint(ansiGray, life))
		}
	} else {
		parts = append(parts, r.p.paint(ansiYellow, "cold"))
	}
	if pc.Misses > 0 {
		parts = append(parts, r.p.paint(ansiRed, fmt.Sprintf("miss %d", pc.Misses)))
	}
	if len(parts) == 0 {
		return ""
	}
	return icon + strings.Join(parts, " ")
}

// cacheColor grades a cache hit ratio, where higher is better — the opposite
// direction from usageColor.
func cacheColor(pct float64) string {
	switch {
	case pct >= 80:
		return ansiGreen
	case pct >= 50:
		return ansiYellow
	default:
		return ansiRed
	}
}

// rateLimitSegments renders the subscription windows, each with its usage and
// how long until it resets. Absent windows are skipped.
func (r *renderer) rateLimitSegments() []string {
	rl := r.in.RateLimits
	if rl == nil {
		return nil
	}
	var segs []string
	add := func(emoji, label string, l *RateLimit) {
		if l == nil {
			return
		}
		s := r.p.paint(usageColor(l.UsedPercentage), formatPercent(l.UsedPercentage))
		// The reset countdown follows the usage with no marker: symbols like
		// ⟳ fall back to a different font in many terminals and render two
		// cells wide, which misaligns the rest of the line.
		if in := formatResetIn(l.ResetsAt, r.now); in != "" {
			s += r.p.paint(ansiGray, " "+in)
		}
		segs = append(segs, r.icon(emoji, "")+r.p.paint(ansiGray, label+" ")+s)
	}
	add("⏳", "5h", rl.FiveHour)
	add("📅", "7d", rl.SevenDay)
	add("💳", "spend", rl.SpendLimit)
	return segs
}

func joinNonEmpty(sep string, parts ...string) string {
	kept := parts[:0:0]
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}
