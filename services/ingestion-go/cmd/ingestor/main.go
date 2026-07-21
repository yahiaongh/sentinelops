package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/config"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/consumer"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/metrics"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/migrate"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()
	metrics.MustRegister()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.New(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("connected to TimescaleDB")

	if err := migrate.Apply(ctx, db.Pool(), logger); err != nil {
		logger.Error("failed to apply schema migration", "error", err)
		os.Exit(1)
	}

	// Prometheus metrics + health endpoint
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	metricsServer := &http.Server{
		Addr:              cfg.MetricsAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("metrics server listening", "addr", cfg.MetricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server error", "error", err)
		}
	}()

	c := consumer.New(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, logger)
	events := make(chan consumer.LogEvent, cfg.BatchSize*2)

	go func() {
		if err := c.Run(ctx, events); err != nil {
			logger.Error("consumer stopped with error", "error", err)
		}
	}()

	// Batching loop: flush on size threshold OR time interval, whichever comes first.
	go runBatcher(ctx, db, events, cfg.BatchSize, cfg.BatchInterval, logger)

	logger.Info("sentinelops ingestion service started",
		"kafka_brokers", cfg.KafkaBrokers,
		"topic", cfg.KafkaTopic,
		"batch_size", cfg.BatchSize,
		"batch_interval", cfg.BatchInterval,
	)

	<-ctx.Done()
	logger.Info("shutdown signal received, draining...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = metricsServer.Shutdown(shutdownCtx)

	// Give the batcher a moment to flush any in-flight batch before exit.
	time.Sleep(500 * time.Millisecond)
	logger.Info("shutdown complete")
}

func runBatcher(
	ctx context.Context,
	db *store.Store,
	events <-chan consumer.LogEvent,
	batchSize int,
	batchInterval time.Duration,
	logger *slog.Logger,
) {
	buf := make([]consumer.LogEvent, 0, batchSize)
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	flush := func() {
		if len(buf) == 0 {
			return
		}
		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.FlushBatch(flushCtx, buf); err != nil {
			logger.Error("batch flush failed", "error", err, "batch_len", len(buf))
		} else {
			logger.Info("batch flushed", "count", len(buf))
		}
		buf = buf[:0]
	}

	for {
		select {
		case e, ok := <-events:
			if !ok {
				flush()
				return
			}
			buf = append(buf, e)
			if len(buf) >= batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		}
	}
}
