package ingest

import "github.com/prometheus/client_golang/prometheus"

type TelemetryMetrics struct {
	MessagesReceived   prometheus.Counter
	MessagesProcessed  prometheus.Counter
	MessagesFailed     prometheus.Counter
	MessagesDuplicate  prometheus.Counter
	BatchFlushTotal    prometheus.Counter
	BatchFlushErrors   prometheus.Counter
	CircuitBreakerOpen prometheus.Gauge
}

func NewTelemetryMetrics(registry *prometheus.Registry) *TelemetryMetrics {
	m := &TelemetryMetrics{
		MessagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "telemetry_messages_received_total",
			Help: "Total number of telemetry messages received from NATS.",
		}),
		MessagesProcessed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "telemetry_messages_processed_total",
			Help: "Total number of telemetry messages successfully processed.",
		}),
		MessagesFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "telemetry_messages_failed_total",
			Help: "Total number of telemetry messages that failed processing.",
		}),
		MessagesDuplicate: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "telemetry_messages_duplicate_total",
			Help: "Total number of duplicate telemetry messages received.",
		}),
		BatchFlushTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "telemetry_batch_flush_total",
			Help: "Total number of VictoriaMetrics batch flushes.",
		}),
		BatchFlushErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "telemetry_batch_flush_errors_total",
			Help: "Total number of VictoriaMetrics batch flush errors.",
		}),
		CircuitBreakerOpen: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "telemetry_circuit_breaker_open",
			Help: "Whether the circuit breaker is open (1) or not (0).",
		}),
	}
	if registry != nil {
		registry.MustRegister(
			m.MessagesReceived,
			m.MessagesProcessed,
			m.MessagesFailed,
			m.MessagesDuplicate,
			m.BatchFlushTotal,
			m.BatchFlushErrors,
			m.CircuitBreakerOpen,
		)
	}
	return m
}
