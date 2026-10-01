package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeSessionFile(t *testing.T, dir, name, body string) {
	t.Helper()
	sessions := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessions, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReadRemoteControlFromSessionFile(t *testing.T) {
	t.Setenv("CLAUDE_CODE_BRIDGE_SESSION_ID", "")
	dir := t.TempDir()
	writeSessionFile(t, dir, "100.json", `{"pid":100,"sessionId":"s-1","bridgeSessionId":"session_old","updatedAt":1}`)
	writeSessionFile(t, dir, "200.json", `{"pid":200,"sessionId":"s-1","bridgeSessionId":"session_new","updatedAt":2}`)
	writeSessionFile(t, dir, "300.json", `{"pid":300,"sessionId":"s-2","bridgeSessionId":null,"updatedAt":3}`)
	writeSessionFile(t, dir, "400.json", `{"pid":400,"sessionId":"s-3","updatedAt":3}`)
	writeSessionFile(t, dir, "500.json", `not json s-1`)

	for _, tc := range []struct {
		session string
		want    remoteControl
	}{
		{"s-1", remoteControl{Known: true, SessionID: "session_new"}}, // newest file wins
		{"s-2", remoteControl{Known: true}},                           // disconnected
		{"s-3", remoteControl{Known: true}},                           // never connected
		{"s-4", remoteControl{}},                                      // no file
		{"", remoteControl{}},
	} {
		if got := readRemoteControl(dir, tc.session); got != tc.want {
			t.Errorf("readRemoteControl(%q) = %+v, want %+v", tc.session, got, tc.want)
		}
	}
}

func TestReadRemoteControlPrefersEnv(t *testing.T) {
	t.Setenv("CLAUDE_CODE_BRIDGE_SESSION_ID", "session_env")
	got := readRemoteControl(t.TempDir(), "s-1")
	if want := (remoteControl{Known: true, SessionID: "session_env"}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestRenderRemoteControlRow(t *testing.T) {
	in := fullInput(t)
	now := time.Unix(900_000, 0)

	out := Render(in, testOptions(), gitInfo{Branch: "main"}, now)
	if strings.Contains(out, "remote-control") {
		t.Errorf("want no remote-control row when the state is unknown\ngot:\n%s", out)
	}

	opt := testOptions()
	opt.Remote = remoteControl{Known: true}
	rows := strings.Split(Render(in, opt, gitInfo{Branch: "main"}, now), "\n")
	if len(rows) != 5 || rows[4] != "remote-control off" {
		t.Errorf("want remote-control off on the last of 5 rows, got %q", rows)
	}

	opt.Remote = remoteControl{Known: true, SessionID: "session_abc"}
	opt.ConfigDir = "/tmp/alt"
	rows = strings.Split(Render(in, opt, gitInfo{Branch: "main"}, now), "\n")
	if last := rows[len(rows)-1]; last != "remote-control on session_abc" {
		t.Errorf("want remote-control on as the bottom row, got %q", last)
	}

	opt.Links = true
	out = Render(in, opt, gitInfo{Branch: "main"}, now)
	if !strings.Contains(out, "\x1b]8;;https://claude.ai/code/session_abc\x1b\\") {
		t.Errorf("want an OSC 8 link to the claude.ai session\ngot:\n%q", out)
	}
}
