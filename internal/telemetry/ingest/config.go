package ingest

import (
	"fmt"
	"time"
)

type Config struct {
	NATSURL     string
	VMURL       string
	DatabaseURL string

	StreamName    string
	ConsumerName  string
	QueueName     string
	DLQStreamName string

	Workers         int
	BatchSize       int
	FlushInterval   time.Duration
	HTTPTimeout     time.Duration
	ShutdownTimeout time.Duration

	CircuitBreaker CircuitBreakerConfig
	Stream         StreamConfig
}

type CircuitBreakerConfig struct {
	FailureThreshold int
	OpenDuration     time.Duration
	HalfOpenMaxReqs  int
}

type StreamConfig struct {
	MaxAge   time.Duration
	MaxBytes int64
	Replicas int
}

func (c Config) Validate() error {
	if c.NATSURL == "" {
		return fmt.Errorf("NATS_URL is required")
	}
	if c.VMURL == "" {
		return fmt.Errorf("VICTORIA_URL is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	return nil
}
