package ingest

import (
	"errors"
	"sync"
	"time"
)

type CircuitState string

const (
	CBClosed   CircuitState = "closed"
	CBOpen     CircuitState = "open"
	CBHalfOpen CircuitState = "half_open"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type CircuitBreaker struct {
	cfg         CircuitBreakerConfig
	mu          sync.Mutex
	state       CircuitState
	failures    int
	openedAt    time.Time
	halfOpenCnt int
}

func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.OpenDuration == 0 {
		cfg.OpenDuration = 30 * time.Second
	}
	if cfg.HalfOpenMaxReqs == 0 {
		cfg.HalfOpenMaxReqs = 1
	}
	return &CircuitBreaker{
		cfg:   cfg,
		state: CBClosed,
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transition()
	return cb.state
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	cb.transition()
	if cb.state == CBOpen {
		cb.mu.Unlock()
		return ErrCircuitOpen
	}
	if cb.state == CBHalfOpen && cb.halfOpenCnt >= cb.cfg.HalfOpenMaxReqs {
		cb.mu.Unlock()
		return ErrCircuitOpen
	}
	if cb.state == CBHalfOpen {
		cb.halfOpenCnt++
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err != nil {
		cb.failures++
		if cb.state == CBHalfOpen {
			cb.state = CBOpen
			cb.openedAt = time.Now()
		} else if cb.failures >= cb.cfg.FailureThreshold {
			cb.state = CBOpen
			cb.openedAt = time.Now()
		}
		return err
	}
	cb.failures = 0
	if cb.state == CBHalfOpen {
		cb.state = CBClosed
		cb.halfOpenCnt = 0
	}
	return nil
}

func (cb *CircuitBreaker) transition() {
	if cb.state == CBOpen && time.Since(cb.openedAt) >= cb.cfg.OpenDuration {
		cb.state = CBHalfOpen
		cb.halfOpenCnt = 0
		cb.failures = 0
	}
}
