package store

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/consumer"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestFlushBatch_EmptySlice_NoOp(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	s := &Store{pool: mock, logger: testLogger()}

	// No expectations set on the mock — if FlushBatch tries to touch the
	// database for an empty slice, ExpectationsWereMet will fail below,
	// proving the early-return short-circuit actually works.
	if err := s.FlushBatch(context.Background(), nil); err != nil {
		t.Errorf("expected nil error for empty slice, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expected zero database interaction for an empty batch, got: %v", err)
	}
}

func TestFlushBatch_AllInvalidTimestamps_NoOp(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	s := &Store{pool: mock, logger: testLogger()}

	events := []consumer.LogEvent{
		{EventID: "a", Timestamp: "not-a-timestamp"},
		{EventID: "b", Timestamp: ""},
	}

	// Every event has an unparsable timestamp, so FlushBatch should skip
	// all of them and never open a transaction — again proven by the mock
	// having zero expectations set.
	if err := s.FlushBatch(context.Background(), events); err != nil {
		t.Errorf("expected nil error when all events have invalid timestamps, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expected zero database interaction when no events are valid, got: %v", err)
	}
}
