package consumer

import "time"

// LogEvent mirrors the JSON schema emitted by the Python log-producer.
type LogEvent struct {
	EventID         string  `json:"event_id"`
	Timestamp       string  `json:"timestamp"`
	Service         string  `json:"service"`
	Endpoint        string  `json:"endpoint"`
	Level           string  `json:"level"`
	StatusCode      int     `json:"status_code"`
	LatencyMs       float64 `json:"latency_ms"`
	Message         string  `json:"message"`
	TraceID         string  `json:"trace_id"`
	AnomalyInjected bool    `json:"anomaly_injected"`
}

func (e LogEvent) ParsedTimestamp() (time.Time, error) {
	return time.Parse(time.RFC3339Nano, e.Timestamp)
}