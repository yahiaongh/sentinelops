package consumer

import (
	"encoding/json"
	"testing"
)

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

func TestLogEvent_JSONRoundTrip(t *testing.T) {
	original := LogEvent{
		EventID:         "abc-123",
		Timestamp:       "2026-07-20T16:30:11.979003225Z",
		Service:         "checkout",
		Endpoint:        "/cart/checkout",
		Level:           "ERROR",
		StatusCode:      500,
		LatencyMs:       342.5,
		Message:         "internal error",
		TraceID:         "trace-xyz",
		AnomalyInjected: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded LogEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch:\ngot:  %+v\nwant: %+v", decoded, original)
	}
}

func TestParsedTimestamp_EmptyString(t *testing.T) {
	e := LogEvent{Timestamp: ""}
	if _, err := e.ParsedTimestamp(); err == nil {
		t.Error("expected an error for empty timestamp, got nil")
	}
}
