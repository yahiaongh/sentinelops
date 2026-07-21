package consumer

import "testing"

func TestParsedTimestamp_Valid(t *testing.T) {
	e := LogEvent{Timestamp: "2026-07-20T16:30:11.979003225Z"}
	ts, err := e.ParsedTimestamp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ts.Year() != 2026 {
		t.Errorf("expected year 2026, got %d", ts.Year())
	}
}

func TestParsedTimestamp_Invalid(t *testing.T) {
	e := LogEvent{Timestamp: "not-a-timestamp"}
	if _, err := e.ParsedTimestamp(); err == nil {
		t.Error("expected an error for invalid timestamp, got nil")
	}
}
