package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRejectsEmptyStdin(t *testing.T) {
	var out bytes.Buffer
	err := run(strings.NewReader(""), &out, testOptions())
	if err == nil {
		t.Fatal("want an error for empty stdin")
	}
	if out.Len() != 0 {
		t.Errorf("nothing should be printed on error, got %q", out.String())
	}
}

func TestRunRejectsInvalidJSON(t *testing.T) {
	var out bytes.Buffer
	if err := run(strings.NewReader("not json"), &out, testOptions()); err == nil {
		t.Fatal("want an error for malformed JSON")
	}
}

func TestRunPrintsOneTrailingNewline(t *testing.T) {
	var out bytes.Buffer
	payload := `{"model":{"display_name":"Opus"},"cwd":"/","context_window":{},"cost":{}}`
	if err := run(strings.NewReader(payload), &out, testOptions()); err != nil {
		t.Fatalf("run: %v", err)
	}
	s := out.String()
	if !strings.HasSuffix(s, "\n") || strings.HasSuffix(s, "\n\n") {
		t.Errorf("want exactly one trailing newline, got %q", s)
	}
	if !strings.Contains(s, "Opus") {
		t.Errorf("want the model name in the output, got %q", s)
	}
}

func TestEnvInt(t *testing.T) {
	t.Setenv("CCSL_TEST_COLS", "120")
	if got := envInt("CCSL_TEST_COLS"); got != 120 {
		t.Errorf("got %d, want 120", got)
	}
	t.Setenv("CCSL_TEST_COLS", "nope")
	if got := envInt("CCSL_TEST_COLS"); got != 0 {
		t.Errorf("got %d, want 0 for an unparseable value", got)
	}
	if got := envInt("CCSL_TEST_UNSET_VAR"); got != 0 {
		t.Errorf("got %d, want 0 for an unset variable", got)
	}
}
