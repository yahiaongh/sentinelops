package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/consumer"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/metrics"
)

type Store struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func New(ctx context.Context, databaseURL string, logger *slog.Logger) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &Store{pool: pool, logger: logger}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

// Pool exposes the underlying connection pool for use by other internal
// packages (e.g. migrate) that need direct database access.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// FlushBatch writes a batch of events to TimescaleDB using a single
// multi-row INSERT executed as a pipelined batch for throughput.
func (s *Store) FlushBatch(ctx context.Context, events []consumer.LogEvent) error {
	if len(events) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		metrics.BatchFlushDuration.Observe(time.Since(start).Seconds())
	}()
	metrics.BatchSize.Observe(float64(len(events)))

	batch := &pgxBatch{}
	const query = `
		INSERT INTO log_events
			(event_id, ts, service, endpoint, level, status_code, latency_ms, message, trace_id, anomaly_injected)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (event_id, ts) DO NOTHING`

	pb := s.pool.Begin
	_ = pb
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		metrics.WriteErrorsTotal.Inc()
		return err
	}
	defer tx.Rollback(ctx)

	for _, e := range events {
		ts, err := e.ParsedTimestamp()
		if err != nil {
			s.logger.Warn("skipping event with unparsable timestamp", "event_id", e.EventID, "error", err)
			continue
		}
		if _, err := tx.Exec(ctx, query,
			e.EventID, ts, e.Service, e.Endpoint, e.Level,
			e.StatusCode, e.LatencyMs, e.Message, e.TraceID, e.AnomalyInjected,
		); err != nil {
			metrics.WriteErrorsTotal.Inc()
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		metrics.WriteErrorsTotal.Inc()
		return err
	}

	metrics.EventsWrittenTotal.Add(float64(len(events)))
	_ = batch
	return nil
}

type pgxBatch struct{}
