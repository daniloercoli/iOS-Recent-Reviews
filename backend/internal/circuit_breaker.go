package internal

import (
	"sync"
	"time"
)

type cbState int

const (
	cbClosed cbState = iota
	cbOpen
	cbHalfOpen
)

type CircuitBreaker struct {
	mu               sync.Mutex
	state            cbState
	failures         int
	failureThreshold int
	openUntil        time.Time
	openCooldown     time.Duration
}

// NewCircuitBreaker create a new CB with thresholds read from config
func NewCircuitBreaker(failureThreshold int, openCooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            cbClosed,
		failureThreshold: failureThreshold,
		openCooldown:     openCooldown,
	}
}

// Allow determines if the request is permitted now
func (c *CircuitBreaker) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.state {
	case cbClosed:
		return true
	case cbOpen:
		if time.Now().After(c.openUntil) {
			// test window
			c.state = cbHalfOpen
			return true
		}
		return false
	case cbHalfOpen:
		// allow few requests (here 1 at a time)
		return true
	default:
		return true
	}
}

// Success resets counter and (if half-open) closes
func (c *CircuitBreaker) Success() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = 0
	if c.state == cbHalfOpen {
		c.state = cbClosed
	}
}

// Failure increments counter and can open circuit
func (c *CircuitBreaker) Failure() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures++
	switch c.state {
	case cbClosed:
		if c.failures >= c.failureThreshold {
			c.state = cbOpen
			c.openUntil = time.Now().Add(c.openCooldown)
		}
	case cbHalfOpen:
		// go in the open state immediately
		c.state = cbOpen
		c.openUntil = time.Now().Add(c.openCooldown)
		c.failures = c.failureThreshold // segnale che è “pieno”
	case cbOpen:
		// stay open
	}
}

// Debug purpose
func (c *CircuitBreaker) State() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.state {
	case cbClosed:
		return "closed"
	case cbOpen:
		return "open"
	case cbHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}
