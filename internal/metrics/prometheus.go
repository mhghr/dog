package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRegistry builds a per-service Prometheus registry with runtime collectors.
func NewRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return registry
}

func Handler(registry *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

type SchedulerMetrics struct {
	JobsPublished prometheus.Counter
	PublishErrors prometheus.Counter
	BatchDuration prometheus.Histogram
}

func NewSchedulerMetrics(registry *prometheus.Registry) *SchedulerMetrics {
	factory := promauto.With(registry)

	return &SchedulerMetrics{
		JobsPublished: factory.NewCounter(prometheus.CounterOpts{
			Name: "scheduler_jobs_published_total",
			Help: "Total number of probe jobs published to the queue.",
		}),
		PublishErrors: factory.NewCounter(prometheus.CounterOpts{
			Name: "scheduler_publish_errors_total",
			Help: "Total number of failures while publishing probe jobs.",
		}),
		BatchDuration: factory.NewHistogram(prometheus.HistogramOpts{
			Name:    "scheduler_batch_duration_seconds",
			Help:    "Duration of scheduler batch runs.",
			Buckets: prometheus.DefBuckets,
		}),
	}
}

type WorkerMetrics struct {
	JobsReceived  prometheus.Counter
	JobsCompleted prometheus.Counter
	JobsFailed    prometheus.Counter
	ProbeDuration *prometheus.HistogramVec
}

func NewWorkerMetrics(registry *prometheus.Registry) *WorkerMetrics {
	factory := promauto.With(registry)

	return &WorkerMetrics{
		JobsReceived: factory.NewCounter(prometheus.CounterOpts{
			Name: "worker_jobs_received_total",
			Help: "Total number of probe jobs received from the queue.",
		}),
		JobsCompleted: factory.NewCounter(prometheus.CounterOpts{
			Name: "worker_jobs_completed_total",
			Help: "Total number of probe jobs completed successfully.",
		}),
		JobsFailed: factory.NewCounter(prometheus.CounterOpts{
			Name: "worker_jobs_failed_total",
			Help: "Total number of probe jobs that failed to process.",
		}),
		ProbeDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "worker_probe_duration_seconds",
			Help:    "Probe execution duration by monitor type and outcome.",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
		}, []string{"type", "success"}),
	}
}

type IngestionMetrics struct {
	ResultsTotal     prometheus.Counter
	DuplicateResults prometheus.Counter
	QueuePendingJobs prometheus.Gauge
}

func NewIngestionMetrics(registry *prometheus.Registry) *IngestionMetrics {
	factory := promauto.With(registry)

	return &IngestionMetrics{
		ResultsTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "ingestion_results_total",
			Help: "Total number of probe results ingested.",
		}),
		DuplicateResults: factory.NewCounter(prometheus.CounterOpts{
			Name: "ingestion_duplicate_results_total",
			Help: "Total number of duplicate probe results rejected by idempotency.",
		}),
		QueuePendingJobs: factory.NewGauge(prometheus.GaugeOpts{
			Name: "queue_pending_jobs",
			Help: "Probe jobs waiting in the queue (lag + pending).",
		}),
	}
}
