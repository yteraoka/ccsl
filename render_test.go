package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func testOptions() Options {
	o := defaultOptions()
	o.Color = false
	o.Emoji = false
	o.Links = false
	return o
}

func fullInput(t *testing.T) *Input {
	t.Helper()
	const payload = `{
	  "cwd": "/home/u/src/app",
	  "session_id": "2fa45908-49bb-4048-b74c-e58d273f075a",
	  "model": {"id": "claude-opus-5", "display_name": "Opus"},
	  "workspace": {"current_dir": "/home/u/src/app", "project_dir": "/home/u/src/app"},
	  "cost": {"total_cost_usd": 1.2345, "total_duration_ms": 4500000, "total_api_duration_ms": 723000},
	  "context_window": {"total_input_tokens": 84500, "total_output_tokens": 1200,
	                     "context_window_size": 200000, "used_percentage": 42.85},
	  "rate_limits": {"five_hour": {"used_percentage": 23.5, "resets_at": 1000000},
	                  "seven_day": {"used_percentage": 91.2, "resets_at": 1200000}},
	  "pr": {"number": 1234, "url": "https://example.com/pr/1234", "review_state": "approved"},
	  "worktree": {"name": "my-feature", "branch": "worktree-my-feature", "original_branch": "main"}
	}`
	var in Input
	if err := json.Unmarshal([]byte(payload), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &in
}

func TestRenderIncludesEveryRequestedField(t *testing.T) {
	t.Setenv("HOME", "/home/u")
	in := fullInput(t)
	git := gitInfo{Branch: "main", Dirty: true, Ahead: 2}
	now := time.Unix(900_000, 0)

	out := Render(in, testOptions(), git, now)

	for _, want := range []string{
		"Opus",         // model
		"~/src/app",    // working directory
		"main✱ ↑2",     // git branch
		"my-feature",   // worktree
		"← main",       // worktree origin branch
		"PR #1234",     // pull request
		"(approved)",   // review state
		"43%",          // context usage
		"(85.7k/200k)", // token counts
		"5h 24%",       // 5-hour limit
		"7d 91%",       // 7-day limit
		"$1.23",        // cost
		"1h15m",        // session duration
		"12m03s",       // API time
		"2fa45908",     // session id
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestRenderThreeLinesByDefault(t *testing.T) {
	in := fullInput(t)
	out := Render(in, testOptions(), gitInfo{Branch: "main"}, time.Unix(900_000, 0))
	rows := strings.Split(out, "\n")
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d:\n%s", len(rows), out)
	}
	if !strings.HasPrefix(rows[2], "id 2fa45908-49bb-4048-b74c-e58d273f075a") {
		t.Errorf("want the full session id leading row 3, got %q", rows[2])
	}
	if !strings.Contains(rows[2], "session 1h15m") || !strings.Contains(rows[2], "api 12m03s") {
		t.Errorf("want the elapsed times on row 3, got %q", rows[2])
	}
	if strings.Contains(rows[1], "session 1h15m") || strings.Contains(rows[1], "api 12m03s") {
		t.Errorf("elapsed times should have left row 2, got %q", rows[1])
	}

	opt := testOptions()
	opt.SingleLine = true
	out = Render(in, opt, gitInfo{Branch: "main"}, time.Unix(900_000, 0))
	if strings.Contains(out, "\n") {
		t.Errorf("one-line mode emitted a newline:\n%s", out)
	}
}

func TestRenderOmitsAbsentOptionalFields(t *testing.T) {
	in := &Input{}
	in.Model.DisplayName = "Sonnet"
	in.CWD = "/tmp"

	out := Render(in, testOptions(), gitInfo{}, time.Now())

	for _, unwanted := range []string{"pr", "worktree", "branch", "5h", "7d"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("output should not mention %q when the field is absent\ngot:\n%s", unwanted, out)
		}
	}
	// Context percentage is unknown before the first API response.
	if !strings.Contains(out, "ctx —") {
		t.Errorf("want an unknown-context placeholder, got:\n%s", out)
	}
}

func TestRenderRateLimitCountdown(t *testing.T) {
	in := fullInput(t)
	// resets_at values are 100000s and 300000s ahead of this instant.
	out := Render(in, testOptions(), gitInfo{}, time.Unix(900_000, 0))
	if !strings.Contains(out, "5h 24% 1d3h") {
		t.Errorf("want the 5-hour countdown after the usage:\n%s", out)
	}
	if !strings.Contains(out, "7d 91% 3d11h") {
		t.Errorf("want the 7-day countdown after the usage:\n%s", out)
	}
}

func TestRenderExpiredRateLimitWindowDropsCountdown(t *testing.T) {
	in := fullInput(t)
	// now is past both resets_at values.
	out := Render(in, testOptions(), gitInfo{}, time.Unix(2_000_000, 0))
	if !strings.Contains(out, "5h 24% \u2502") {
		t.Errorf("want the usage alone, with no countdown:\n%s", out)
	}
	if !strings.Contains(out, "7d 91%") {
		t.Errorf("usage should still render:\n%s", out)
	}
}

func TestFirstLineShrinksDirectoryToFitColumns(t *testing.T) {
	t.Setenv("HOME", "/home/u")
	in := fullInput(t)
	in.Workspace.CurrentDir = "/home/u/a/very/deeply/nested/project/directory"

	opt := testOptions()
	opt.Columns = 60
	out := Render(in, opt, gitInfo{Branch: "main"}, time.Unix(900_000, 0))
	first := strings.SplitN(out, "\n", 2)[0]

	if displayWidth(first) > opt.Columns {
		t.Errorf("first line is %d cells wide, want <= %d:\n%s", displayWidth(first), opt.Columns, first)
	}
	if !strings.Contains(first, "…") {
		t.Errorf("expected an elided path, got:\n%s", first)
	}
}

func TestRenderMarksOversizedContext(t *testing.T) {
	in := fullInput(t)
	in.Exceeds200kTokens = true
	if !strings.Contains(Render(in, testOptions(), gitInfo{}, time.Now()), "⚠") {
		t.Error("want a warning marker when exceeds_200k_tokens is set")
	}
}

func TestRenderMergeRequest(t *testing.T) {
	in := fullInput(t)
	in.PR.Kind = "mr"
	out := Render(in, testOptions(), gitInfo{}, time.Now())
	if !strings.Contains(out, "MR !1234") {
		t.Errorf("want GitLab merge request label, got:\n%s", out)
	}
}

func TestPRLinkUsesOSC8WhenEnabled(t *testing.T) {
	in := fullInput(t)
	opt := testOptions()
	opt.Links = true
	out := Render(in, opt, gitInfo{}, time.Now())
	if !strings.Contains(out, "\033]8;;https://example.com/pr/1234\033\\") {
		t.Errorf("want an OSC 8 hyperlink, got:\n%q", out)
	}
}

func TestGitWorktreeFallsBackToWorkspaceField(t *testing.T) {
	in := fullInput(t)
	in.Worktree = nil
	in.Workspace.GitWorktree = "feature-xyz"
	if !strings.Contains(Render(in, testOptions(), gitInfo{}, time.Now()), "worktree feature-xyz") {
		t.Error("want workspace.git_worktree rendered when worktree.* is absent")
	}
}

func withCache(t *testing.T, pc *PromptCache) *Input {
	t.Helper()
	in := fullInput(t)
	in.PromptCache = pc
	return in
}

func TestRenderCacheWarm(t *testing.T) {
	now := time.Unix(900_000, 0)
	expires := now.Add(42 * time.Minute).Unix()
	in := withCache(t, &PromptCache{
		Warm: true, CachingObserved: true, TTL: "1h",
		ExpiresAt: &expires, HitRatio: ptr(0.91), Requests: 14,
	})
	out := Render(in, testOptions(), gitInfo{}, now)
	if !strings.Contains(out, "cache 91% 42m/1h") {
		t.Errorf("want hit ratio and remaining lifetime, got:\n%s", out)
	}
}

func TestRenderCacheCold(t *testing.T) {
	in := withCache(t, &PromptCache{
		Warm: false, CachingObserved: true, TTL: "5m",
		HitRatio: ptr(0.42), Misses: 2,
	})
	out := Render(in, testOptions(), gitInfo{}, time.Unix(900_000, 0))
	if !strings.Contains(out, "cache 42% cold miss 2") {
		t.Errorf("want a cold cache with its miss count, got:\n%s", out)
	}
}

func TestRenderCacheNotObserved(t *testing.T) {
	in := withCache(t, &PromptCache{Warm: false, CachingObserved: false})
	out := Render(in, testOptions(), gitInfo{}, time.Unix(900_000, 0))
	if !strings.Contains(out, "cache off") {
		t.Errorf("want the cache reported as off, got:\n%s", out)
	}
}

func TestRenderCacheAbsent(t *testing.T) {
	in := fullInput(t) // no prompt_cache in the payload
	if strings.Contains(Render(in, testOptions(), gitInfo{}, time.Now()), "cache") {
		t.Error("the cache segment should be omitted before the first API response")
	}
}

func TestRenderCacheWithoutHitRatio(t *testing.T) {
	// hit_ratio is null while the token counts are all zero.
	in := withCache(t, &PromptCache{Warm: true, CachingObserved: true, TTL: "5m"})
	out := Render(in, testOptions(), gitInfo{}, time.Unix(900_000, 0))
	if !strings.Contains(out, "cache 5m") {
		t.Errorf("want the lifetime alone when the ratio is unknown, got:\n%s", out)
	}
}

func TestCacheColor(t *testing.T) {
	// Higher is better, the opposite of usageColor.
	tests := []struct {
		pct  float64
		want string
	}{
		{95, ansiGreen},
		{80, ansiGreen},
		{60, ansiYellow},
		{50, ansiYellow},
		{10, ansiRed},
	}
	for _, tt := range tests {
		if got := cacheColor(tt.pct); got != tt.want {
			t.Errorf("cacheColor(%v) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}

func ptr[T any](v T) *T { return &v }
