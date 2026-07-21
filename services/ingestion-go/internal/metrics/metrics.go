package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	EventsConsumedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinelops_ingestion_events_consumed_total",
			Help: "Total number of log events consumed from Kafka, by service.",
		},
		[]string{"service"},
	)

	EventsWrittenTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sentinelops_ingestion_events_written_total",
			Help: "Total number of log events successfully written to TimescaleDB.",
		},
	)

	WriteErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sentinelops_ingestion_write_errors_total",
			Help: "Total number of failed batch writes to TimescaleDB.",
		},
	)

	BatchFlushDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "sentinelops_ingestion_batch_flush_duration_seconds",
			Help:    "Time taken to flush a batch of events to TimescaleDB.",
			Buckets: prometheus.DefBuckets,
		},
	)

	BatchSize = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "sentinelops_ingestion_batch_size",
			Help:    "Number of events per flushed batch.",
			Buckets: []float64{1, 10, 25, 50, 100, 200, 500},
		},
	)

	ConsumerLagGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sentinelops_ingestion_kafka_fetch_errors_total",
			Help: "Total Kafka fetch errors encountered by the consumer.",
		},
	)
)

func MustRegister() {
	prometheus.MustRegister(
		EventsConsumedTotal,
		EventsWrittenTotal,
		WriteErrorsTotal,
		BatchFlushDuration,
		BatchSize,
		ConsumerLagGauge,
	)
}
