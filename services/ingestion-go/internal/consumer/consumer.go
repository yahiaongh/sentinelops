package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/yahiaongh/sentinelops/services/ingestion-go/internal/metrics"
)

type Consumer struct {
	reader *kafka.Reader
	logger *slog.Logger
}

func New(brokers []string, topic, groupID string, logger *slog.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})
	return &Consumer{reader: reader, logger: logger}
}

// Run reads messages continuously and pushes parsed events onto out.
// It blocks until ctx is cancelled, then closes the reader.
func (c *Consumer) Run(ctx context.Context, out chan<- LogEvent) error {
	defer close(out)
	defer c.reader.Close()

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.logger.Info("consumer shutting down")
				return nil
			}
			metrics.ConsumerLagGauge.Inc()
			c.logger.Error("fetch message failed", "error", err)
			continue
		}

		var event LogEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("failed to unmarshal event, skipping", "error", err)
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("commit failed after skip", "error", err)
			}
			continue
		}

		metrics.EventsConsumedTotal.WithLabelValues(event.Service).Inc()

		select {
		case out <- event:
		case <-ctx.Done():
			return nil
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("commit failed", "error", err)
		}
	}
}