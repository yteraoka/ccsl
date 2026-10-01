package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

// remoteControl is the Remote Control (claude.ai / mobile bridge) state of the
// session. The status line payload does not carry it, so it is recovered from
// what Claude Code itself records about the running process.
type remoteControl struct {
	// Known is false when the state could not be determined; the status line
	// then omits the segment instead of guessing "off".
	Known bool
	// SessionID is the bridge session ("session_...") while Remote Control is
	// connected, and empty when it is off.
	SessionID string
}

// remoteSessionURL is where claude.ai serves a Remote Control session.
func remoteSessionURL(id string) string {
	return "https://claude.ai/code/" + id
}

// sessionPIDFile is the subset of <config>/sessions/<pid>.json this tool reads.
// Claude Code writes one per running process and sets bridgeSessionId while
// Remote Control is connected, resetting it to null when it disconnects.
type sessionPIDFile struct {
	SessionID       string  `json:"sessionId"`
	BridgeSessionID *string `json:"bridgeSessionId"`
	UpdatedAt       int64   `json:"updatedAt"`
}

// readRemoteControl looks up the Remote Control state for sessionID.
//
// Claude Code exports CLAUDE_CODE_BRIDGE_SESSION_ID to its own environment
// while connected, so that is trusted first. Otherwise the per-process session
// files under configDir are scanned for this session; a resumed session can
// leave files from earlier processes behind, so the most recently updated one
// wins.
func readRemoteControl(configDir, sessionID string) remoteControl {
	if id := os.Getenv("CLAUDE_CODE_BRIDGE_SESSION_ID"); id != "" {
		return remoteControl{Known: true, SessionID: id}
	}
	if sessionID == "" {
		return remoteControl{}
	}
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return remoteControl{}
		}
		configDir = filepath.Join(home, ".claude")
	}
	paths, err := filepath.Glob(filepath.Join(configDir, "sessions", "*.json"))
	if err != nil {
		return remoteControl{}
	}

	var (
		best  sessionPIDFile
		found bool
	)
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil || !bytes.Contains(data, []byte(sessionID)) {
			continue
		}
		var f sessionPIDFile
		if json.Unmarshal(data, &f) != nil || f.SessionID != sessionID {
			continue
		}
		if !found || f.UpdatedAt > best.UpdatedAt {
			best, found = f, true
		}
	}
	if !found {
		return remoteControl{}
	}
	rc := remoteControl{Known: true}
	if best.BridgeSessionID != nil {
		rc.SessionID = *best.BridgeSessionID
	}
	return rc
}
