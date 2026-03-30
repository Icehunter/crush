package auto

import (
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ErrorClass categorizes errors for retry strategy selection.
type ErrorClass int

const (
	// ErrClassUnknown is the default for errors that don't match any pattern.
	ErrClassUnknown ErrorClass = iota
	// ErrClassRateLimit indicates an API rate limit was hit.
	ErrClassRateLimit
	// ErrClassServer indicates a provider server error (5xx).
	ErrClassServer
	// ErrClassNetwork indicates a transient network error.
	ErrClassNetwork
	// ErrClassPermanent indicates an error that will not resolve on retry.
	ErrClassPermanent
)

func (c ErrorClass) String() string {
	switch c {
	case ErrClassRateLimit:
		return "rate_limit"
	case ErrClassServer:
		return "server"
	case ErrClassNetwork:
		return "network"
	case ErrClassPermanent:
		return "permanent"
	default:
		return "unknown"
	}
}

// Retryable returns true for error classes where a retry may succeed.
func (c ErrorClass) Retryable() bool {
	return c != ErrClassPermanent
}

// Regex patterns for error classification.
var (
	rateLimitPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)rate.?limit`),
		regexp.MustCompile(`(?i)too many requests`),
		regexp.MustCompile(`(?i)429`),
		regexp.MustCompile(`(?i)throttl`),
		regexp.MustCompile(`(?i)overloaded`),
	}
	serverPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)5\d{2}\b`),
		regexp.MustCompile(`(?i)internal server error`),
		regexp.MustCompile(`(?i)bad gateway`),
		regexp.MustCompile(`(?i)service unavailable`),
		regexp.MustCompile(`(?i)gateway timeout`),
	}
	networkPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)connection refused`),
		regexp.MustCompile(`(?i)connection reset`),
		regexp.MustCompile(`(?i)no such host`),
		regexp.MustCompile(`(?i)dial tcp`),
		regexp.MustCompile(`(?i)timeout`),
		regexp.MustCompile(`(?i)tls handshake`),
		regexp.MustCompile(`(?i)EOF`),
		regexp.MustCompile(`(?i)broken pipe`),
	}
	permanentPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)invalid.?api.?key`),
		regexp.MustCompile(`(?i)authentication`),
		regexp.MustCompile(`(?i)unauthorized`),
		regexp.MustCompile(`(?i)forbidden`),
		regexp.MustCompile(`(?i)not found`),
		regexp.MustCompile(`(?i)invalid.?model`),
		regexp.MustCompile(`(?i)billing`),
		regexp.MustCompile(`(?i)quota exceeded`),
	}
)

// ClassifyError inspects an error's message and returns the best-matching
// ErrorClass. Patterns are checked in priority order: permanent first
// (to avoid retrying auth errors), then rate-limit, server, network.
func ClassifyError(err error) ErrorClass {
	if err == nil {
		return ErrClassUnknown
	}
	msg := err.Error()

	for _, p := range permanentPatterns {
		if p.MatchString(msg) {
			return ErrClassPermanent
		}
	}
	for _, p := range rateLimitPatterns {
		if p.MatchString(msg) {
			return ErrClassRateLimit
		}
	}
	for _, p := range serverPatterns {
		if p.MatchString(msg) {
			return ErrClassServer
		}
	}
	for _, p := range networkPatterns {
		if p.MatchString(msg) {
			return ErrClassNetwork
		}
	}

	// Heuristic: if the message contains "error" with a status code pattern,
	// classify based on the code range.
	if strings.Contains(strings.ToLower(msg), "status") {
		return ErrClassServer
	}

	return ErrClassUnknown
}

// BackoffConfig defines retry limits and timing per error class.
type BackoffConfig struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
	MaxRetries int
}

// DefaultBackoffConfig returns the backoff configuration for an error class.
func DefaultBackoffConfig(class ErrorClass) BackoffConfig {
	switch class {
	case ErrClassRateLimit:
		return BackoffConfig{
			BaseDelay:  2 * time.Second,
			MaxDelay:   60 * time.Second,
			MaxRetries: 10,
		}
	case ErrClassServer:
		return BackoffConfig{
			BaseDelay:  2 * time.Second,
			MaxDelay:   30 * time.Second,
			MaxRetries: 5,
		}
	case ErrClassNetwork:
		return BackoffConfig{
			BaseDelay:  1 * time.Second,
			MaxDelay:   15 * time.Second,
			MaxRetries: 5,
		}
	case ErrClassPermanent:
		return BackoffConfig{
			BaseDelay:  0,
			MaxDelay:   0,
			MaxRetries: 0,
		}
	default:
		return BackoffConfig{
			BaseDelay:  2 * time.Second,
			MaxDelay:   30 * time.Second,
			MaxRetries: 3,
		}
	}
}

// RetryState tracks exponential backoff for a sequence of classified errors.
// It resets when a dispatch succeeds.
type RetryState struct {
	mu           sync.Mutex
	consecutives int
	lastClass    ErrorClass
}

// NewRetryState creates a fresh retry state.
func NewRetryState() *RetryState {
	return &RetryState{}
}

// RecordFailure records an error and returns the delay to wait before the
// next attempt and whether retry is allowed. If not allowed, the caller
// should pause the engine.
func (r *RetryState) RecordFailure(err error) (delay time.Duration, retryable bool) {
	class := ClassifyError(err)
	cfg := DefaultBackoffConfig(class)

	r.mu.Lock()
	defer r.mu.Unlock()

	if class == r.lastClass {
		r.consecutives++
	} else {
		r.consecutives = 1
		r.lastClass = class
	}

	if !class.Retryable() || (cfg.MaxRetries > 0 && r.consecutives > cfg.MaxRetries) {
		return 0, false
	}

	// Exponential backoff: base * 2^(attempt-1), capped at max.
	exp := math.Pow(2, float64(r.consecutives-1))
	delay = time.Duration(float64(cfg.BaseDelay) * exp)
	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}

	return delay, true
}

// RecordSuccess resets the retry state after a successful dispatch.
func (r *RetryState) RecordSuccess() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.consecutives = 0
	r.lastClass = ErrClassUnknown
}

// Consecutives returns the current consecutive failure count.
func (r *RetryState) Consecutives() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.consecutives
}

// LastClass returns the most recent error class.
func (r *RetryState) LastClass() ErrorClass {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastClass
}
