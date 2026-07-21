//go:build integration

package store

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/consumer"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/migrate"
)

// These tests require a live TimescaleDB instance. Run with:
//   TEST_DATABASE_URL=postgres://sentinelops:devpassword@localhost:5432/sentinelops \
//     go test -tags=integration ./internal/store/...

func TestFlushBatch_WritesAndDeduplicates(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := New(ctx, dbURL, logger)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer s.Close()

	if err := migrate.Apply(ctx, s.Pool(), logger); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	eventID := uuid.NewString()
	event := consumer.LogEvent{
		EventID:         eventID,
		Timestamp:       time.Now().UTC().Format(time.RFC3339Nano),
		Service:         "test-service",
		Endpoint:        "/test",
		Level:           "INFO",
		StatusCode:      200,
		LatencyMs:       12.3,
		Message:         "integration test event",
		TraceID:         uuid.NewString(),
		AnomalyInjected: false,
	}

	// First write should succeed.
	if err := s.FlushBatch(ctx, []consumer.LogEvent{event}); err != nil {
		t.Fatalf("first flush failed: %v", err)
	}

	// Second write of the same event_id+ts should be a no-op (ON CONFLICT DO NOTHING),
	// not an error — this proves idempotency under at-least-once Kafka delivery.
	if err := s.FlushBatch(ctx, []consumer.LogEvent{event}); err != nil {
		t.Fatalf("duplicate flush should not error, got: %v", err)
	}

	var count int
	row := s.Pool().QueryRow(ctx, "SELECT count(*) FROM log_events WHERE event_id = $1", eventID)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("failed to query row count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 row after duplicate flush, got %d", count)
	}
}

func TestFlushBatch_EmptySlice(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	s, err := New(ctx, dbURL, logger)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer s.Close()

	if err := s.FlushBatch(ctx, nil); err != nil {
		t.Errorf("expected nil error on empty batch, got: %v", err)
	}
}
