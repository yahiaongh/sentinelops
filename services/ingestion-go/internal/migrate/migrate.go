package migrate

import (
	"context"
	_ "embed"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed 001_init_hypertable.sql
var initSchema string

// Apply runs the embedded schema migration against the target database.
// It is idempotent: every statement uses IF NOT EXISTS / ON CONFLICT-style
// guards, so it's safe to run on every service startup, whether the schema
// already exists or the volume is brand new.
//
// Takes a concrete *pgxpool.Pool (rather than store.PgxPool) specifically to
// avoid an import cycle: store's integration tests import migrate to apply
// the schema, so migrate cannot import store back.
func Apply(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	logger.Info("applying database schema migration")
	if _, err := pool.Exec(ctx, initSchema); err != nil {
		return err
	}
	logger.Info("schema migration applied successfully")
	return nil
}
