package store

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/consumer"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/metrics"
)

// PgxPool is the subset of *pgxpool.Pool's behavior that Store depends on.
// Defined as an interface (rather than using *pgxpool.Pool directly) so
// tests can substitute a mock pool without needing a live database.
type PgxPool interface {
	Ping(ctx context.Context) error
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Close()
}

type Store struct {
	pool   PgxPool
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
// packages (e.g. migrate) that need direct database access. Returns the
// concrete *pgxpool.Pool (not the PgxPool interface) since callers like
// migrate.Apply need the real type, and s.pool is always a genuine
// *pgxpool.Pool outside of unit tests.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool.(*pgxpool.Pool)
}

const insertLogEventQuery = `
	INSERT INTO log_events
		(event_id, ts, service, endpoint, level, status_code, latency_ms, message, trace_id, anomaly_injected)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT (event_id, ts) DO NOTHING`

// FlushBatch writes a batch of events to TimescaleDB using pgx's pipelined
// Batch API: all inserts are sent over the wire in a single round trip
// instead of N sequential Exec calls, which is the main throughput win over
// naive row-by-row inserts while still preserving per-row ON CONFLICT
// idempotency (unlike COPY, which can't express that).
func (s *Store) FlushBatch(ctx context.Context, events []consumer.LogEvent) error {
	if len(events) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		metrics.BatchFlushDuration.Observe(time.Since(start).Seconds())
	}()
	metrics.BatchSize.Observe(float64(len(events)))

	batch := &pgx.Batch{}
	validCount := 0
	for _, e := range events {
		ts, err := e.ParsedTimestamp()
		if err != nil {
			s.logger.Warn("skipping event with unparsable timestamp", "event_id", e.EventID, "error", err)
			continue
		}
		batch.Queue(insertLogEventQuery,
			e.EventID, ts, e.Service, e.Endpoint, e.Level,
			e.StatusCode, e.LatencyMs, e.Message, e.TraceID, e.AnomalyInjected,
		)
		validCount++
	}

	if validCount == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		metrics.WriteErrorsTotal.Inc()
		return err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Warn("transaction rollback failed", "error", err)
		}
	}()

	results := tx.SendBatch(ctx, batch)
	for i := 0; i < validCount; i++ {
		if _, err := results.Exec(); err != nil {
			metrics.WriteErrorsTotal.Inc()
			_ = results.Close()
			return err
		}
	}
	if err := results.Close(); err != nil {
		metrics.WriteErrorsTotal.Inc()
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		metrics.WriteErrorsTotal.Inc()
		return err
	}

	metrics.EventsWrittenTotal.Add(float64(validCount))
	return nil
}
