package main

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// gitTimeout caps how long the status line waits on git. The command re-runs
// on every assistant message, so a slow repository must never stall it.
const gitTimeout = 400 * time.Millisecond

type gitInfo struct {
	Branch string
	Dirty  bool
	Ahead  int
	Behind int
}

// readGit collects branch and dirtiness for dir. Every failure degrades to an
// empty value: outside a repository the status line simply omits the segment.
func readGit(dir string, wantDirty bool) gitInfo {
	var info gitInfo
	if dir == "" {
		return info
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	info.Branch = gitBranch(ctx, dir)
	if info.Branch == "" {
		return info
	}
	if wantDirty {
		if out, err := git(ctx, dir, "status", "--porcelain", "--untracked-files=normal"); err == nil {
			info.Dirty = strings.TrimSpace(out) != ""
		}
		if out, err := git(ctx, dir, "rev-list", "--left-right", "--count", "@{upstream}...HEAD"); err == nil {
			if f := strings.Fields(out); len(f) == 2 {
				info.Behind = atoi(f[0])
				info.Ahead = atoi(f[1])
			}
		}
	}
	return info
}

func gitBranch(ctx context.Context, dir string) string {
	out, err := git(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(out)
	if branch != "HEAD" {
		return branch
	}
	// Detached HEAD: show the short commit instead of a bare "HEAD".
	if out, err := git(ctx, dir, "rev-parse", "--short", "HEAD"); err == nil {
		return "@" + strings.TrimSpace(out)
	}
	return branch
}

func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(cmd.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	return string(out), err
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
