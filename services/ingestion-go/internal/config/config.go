package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	KafkaBrokers  []string
	KafkaTopic    string
	KafkaGroupID  string
	DatabaseURL   string
	BatchSize     int
	BatchInterval time.Duration
	MetricsAddr   string
}

func Load() Config {
	return Config{
		KafkaBrokers:  []string{getEnv("KAFKA_BROKERS", "redpanda:9092")},
		KafkaTopic:    getEnv("KAFKA_TOPIC", "service-logs"),
		KafkaGroupID:  getEnv("KAFKA_GROUP_ID", "sentinelops-ingestor"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://sentinelops:devpassword@timescaledb:5432/sentinelops"),
		BatchSize:     getEnvInt("BATCH_SIZE", 100),
		BatchInterval: getEnvDuration("BATCH_INTERVAL_MS", 500) * time.Millisecond,
		MetricsAddr:   getEnv("METRICS_ADDR", ":9100"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallbackMs int) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n)
		}
	}
	return time.Duration(fallbackMs)
}
