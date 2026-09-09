// Command ccsl renders a Claude Code status line.
//
// Claude Code pipes session JSON to this command's stdin on every update; the
// command prints the status line to stdout. See:
// https://code.claude.com/docs/en/statusline.md
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	opt := defaultOptions()
	var (
		noColor    = flag.Bool("no-color", false, "disable ANSI colors")
		noEmoji    = flag.Bool("no-emoji", false, "use text labels instead of emoji")
		noLinks    = flag.Bool("no-links", false, "disable OSC 8 clickable links")
		noGitDirty = flag.Bool("no-git-status", false, "skip the dirty / ahead-behind git lookup")
		oneLine    = flag.Bool("one-line", false, "render everything on a single line")
		barWidth   = flag.Int("bar-width", opt.BarWidth, "width of the context usage bar")
		showVer    = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("ccsl", version)
		return
	}

	opt.Color = !*noColor && os.Getenv("NO_COLOR") == ""
	opt.Emoji = !*noEmoji
	// Hyperlinks are independent of color: a terminal with colors turned off
	// can still make the PR clickable.
	opt.Links = !*noLinks
	opt.GitDirty = !*noGitDirty
	opt.SingleLine = *oneLine
	opt.BarWidth = *barWidth
	opt.Columns = envInt("COLUMNS")

	if err := run(os.Stdin, os.Stdout, opt); err != nil {
		fmt.Fprintln(os.Stderr, "ccsl:", err)
		os.Exit(1)
	}
}

func run(stdin io.Reader, stdout io.Writer, opt Options) error {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	if len(data) == 0 {
		return errors.New("no JSON on stdin (this command is meant to be run by Claude Code)")
	}

	var in Input
	if err := json.Unmarshal(data, &in); err != nil {
		return fmt.Errorf("parsing status line JSON: %w", err)
	}

	git := readGit(in.dir(), opt.GitDirty)
	_, err = fmt.Fprintln(stdout, Render(&in, opt, git, time.Now()))
	return err
}

// envInt reads a non-negative integer environment variable, returning 0 when
// it is unset or unparseable. Claude Code sets COLUMNS before running us.
func envInt(name string) int {
	n, err := strconv.Atoi(os.Getenv(name))
	if err != nil || n < 0 {
		return 0
	}
	return n
}
