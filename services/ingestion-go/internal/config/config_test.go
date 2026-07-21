package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)

	cfg := Load()

	if cfg.KafkaTopic != "service-logs" {
		t.Errorf("expected default topic 'service-logs', got %q", cfg.KafkaTopic)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("expected default batch size 100, got %d", cfg.BatchSize)
	}
	if cfg.BatchInterval != 500*time.Millisecond {
		t.Errorf("expected default batch interval 500ms, got %v", cfg.BatchInterval)
	}
	if cfg.MetricsAddr != ":9100" {
		t.Errorf("expected default metrics addr ':9100', got %q", cfg.MetricsAddr)
	}
}

func TestLoad_Overrides(t *testing.T) {
	clearEnv(t)

	t.Setenv("KAFKA_TOPIC", "custom-topic")
	t.Setenv("BATCH_SIZE", "250")
	t.Setenv("BATCH_INTERVAL_MS", "1000")

	cfg := Load()

	if cfg.KafkaTopic != "custom-topic" {
		t.Errorf("expected overridden topic 'custom-topic', got %q", cfg.KafkaTopic)
	}
	if cfg.BatchSize != 250 {
		t.Errorf("expected overridden batch size 250, got %d", cfg.BatchSize)
	}
	if cfg.BatchInterval != 1000*time.Millisecond {
		t.Errorf("expected overridden batch interval 1000ms, got %v", cfg.BatchInterval)
	}
}

func TestGetEnvInt_InvalidFallsBackToDefault(t *testing.T) {
	clearEnv(t)
	t.Setenv("BATCH_SIZE", "not-a-number")

	cfg := Load()

	if cfg.BatchSize != 100 {
		t.Errorf("expected fallback to default 100 on invalid int, got %d", cfg.BatchSize)
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_GROUP_ID",
		"DATABASE_URL", "BATCH_SIZE", "BATCH_INTERVAL_MS", "METRICS_ADDR",
	}
	for _, k := range keys {
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("failed to unset %s: %v", k, err)
		}
	}
}
