package internal

import (
	"encoding/json"
	"io"
	"time"
)

type AppConfig struct {
	AppID   string `json:"appId"`
	Country string `json:"country"`
}

type CircuitBreakerConfig struct {
	FailureThreshold    int `json:"failureThreshold"`    // default 3
	OpenCooldownSeconds int `json:"openCooldownSeconds"` // default 60
}

type Config struct {
	PollIntervalMinutes int                  `json:"pollIntervalMinutes"`
	WebhookURL          string               `json:"webhookUrl"` // could be empty (disabled)
	CircuitBreaker      CircuitBreakerConfig `json:"circuitBreaker"`
	Apps                []AppConfig          `json:"apps"`
}

func ParseConfig(r io.Reader) (*Config, error) {
	var c Config
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return nil, err
	}
	if c.PollIntervalMinutes <= 0 {
		c.PollIntervalMinutes = 15
	}
	if c.CircuitBreaker.FailureThreshold <= 0 {
		c.CircuitBreaker.FailureThreshold = 3
	}
	if c.CircuitBreaker.OpenCooldownSeconds <= 0 {
		c.CircuitBreaker.OpenCooldownSeconds = 60
	}
	return &c, nil
}

type Review struct {
	ID          string    `json:"id"`
	AppID       string    `json:"appId"`
	Country     string    `json:"country"`
	Author      string    `json:"author"`
	Rating      int       `json:"rating"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	SubmittedAt time.Time `json:"submittedAt"` // UTC
}

type StateEntry struct {
	SeenIDs  []string  `json:"seenIds"`
	LastPoll time.Time `json:"lastPoll"`
}

type State struct {
	// key: appId-country
	Entries map[string]*StateEntry `json:"entries"`
}
