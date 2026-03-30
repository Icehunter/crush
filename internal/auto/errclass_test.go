package auto

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClassifyError_RateLimit(t *testing.T) {
	t.Parallel()
	tests := []string{
		"rate limit exceeded",
		"429 Too Many Requests",
		"Request was throttled",
		"API overloaded, please retry",
	}
	for _, msg := range tests {
		require.Equal(t, ErrClassRateLimit, ClassifyError(errors.New(msg)), msg)
	}
}

func TestClassifyError_Server(t *testing.T) {
	t.Parallel()
	tests := []string{
		"500 Internal Server Error",
		"502 Bad Gateway",
		"503 Service Unavailable",
		"504 Gateway Timeout",
	}
	for _, msg := range tests {
		require.Equal(t, ErrClassServer, ClassifyError(errors.New(msg)), msg)
	}
}

func TestClassifyError_Network(t *testing.T) {
	t.Parallel()
	tests := []string{
		"dial tcp: connection refused",
		"connection reset by peer",
		"no such host",
		"TLS handshake failed",
		"unexpected EOF",
		"write: broken pipe",
	}
	for _, msg := range tests {
		require.Equal(t, ErrClassNetwork, ClassifyError(errors.New(msg)), msg)
	}
}

func TestClassifyError_Permanent(t *testing.T) {
	t.Parallel()
	tests := []string{
		"invalid api key provided",
		"authentication failed",
		"401 Unauthorized",
		"403 Forbidden",
		"invalid model specified",
		"billing issue: payment required",
	}
	for _, msg := range tests {
		require.Equal(t, ErrClassPermanent, ClassifyError(errors.New(msg)), msg)
	}
}

func TestClassifyError_Unknown(t *testing.T) {
	t.Parallel()
	require.Equal(t, ErrClassUnknown, ClassifyError(errors.New("something weird happened")))
	require.Equal(t, ErrClassUnknown, ClassifyError(nil))
}

func TestErrorClass_Retryable(t *testing.T) {
	t.Parallel()
	require.True(t, ErrClassRateLimit.Retryable())
	require.True(t, ErrClassServer.Retryable())
	require.True(t, ErrClassNetwork.Retryable())
	require.True(t, ErrClassUnknown.Retryable())
	require.False(t, ErrClassPermanent.Retryable())
}

func TestRetryState_ExponentialBackoff(t *testing.T) {
	t.Parallel()
	rs := NewRetryState()
	err := errors.New("502 Bad Gateway")

	delay1, ok := rs.RecordFailure(err)
	require.True(t, ok)
	require.Equal(t, 2*time.Second, delay1) // base * 2^0

	delay2, ok := rs.RecordFailure(err)
	require.True(t, ok)
	require.Equal(t, 4*time.Second, delay2) // base * 2^1

	delay3, ok := rs.RecordFailure(err)
	require.True(t, ok)
	require.Equal(t, 8*time.Second, delay3) // base * 2^2
}

func TestRetryState_MaxDelayCap(t *testing.T) {
	t.Parallel()
	rs := NewRetryState()
	err := errors.New("502 Bad Gateway")

	// Burn through attempts to hit the cap.
	var lastDelay time.Duration
	for i := 0; i < 5; i++ {
		lastDelay, _ = rs.RecordFailure(err)
	}
	require.LessOrEqual(t, lastDelay, 30*time.Second)
}

func TestRetryState_PermanentNotRetryable(t *testing.T) {
	t.Parallel()
	rs := NewRetryState()
	err := errors.New("invalid api key provided")

	_, ok := rs.RecordFailure(err)
	require.False(t, ok, "permanent errors should not be retryable")
}

func TestRetryState_MaxRetriesExhausted(t *testing.T) {
	t.Parallel()
	rs := NewRetryState()
	err := errors.New("502 Bad Gateway")
	cfg := DefaultBackoffConfig(ErrClassServer)

	for i := 0; i < cfg.MaxRetries; i++ {
		_, ok := rs.RecordFailure(err)
		require.True(t, ok, "attempt %d should be retryable", i+1)
	}

	_, ok := rs.RecordFailure(err)
	require.False(t, ok, "should stop after max retries")
}

func TestRetryState_ResetOnSuccess(t *testing.T) {
	t.Parallel()
	rs := NewRetryState()
	err := errors.New("502 Bad Gateway")

	rs.RecordFailure(err)
	rs.RecordFailure(err)
	require.Equal(t, 2, rs.Consecutives())

	rs.RecordSuccess()
	require.Equal(t, 0, rs.Consecutives())

	// After reset, backoff restarts at base.
	delay, ok := rs.RecordFailure(err)
	require.True(t, ok)
	require.Equal(t, 2*time.Second, delay)
}

func TestRetryState_ClassChange(t *testing.T) {
	t.Parallel()
	rs := NewRetryState()

	rs.RecordFailure(errors.New("502 Bad Gateway"))
	require.Equal(t, ErrClassServer, rs.LastClass())
	require.Equal(t, 1, rs.Consecutives())

	// Different class resets the count.
	delay, ok := rs.RecordFailure(errors.New("rate limit exceeded"))
	require.True(t, ok)
	require.Equal(t, ErrClassRateLimit, rs.LastClass())
	require.Equal(t, 1, rs.Consecutives())
	require.Equal(t, 2*time.Second, delay) // Reset to base.
}

func TestErrorClass_String(t *testing.T) {
	t.Parallel()
	require.Equal(t, "rate_limit", ErrClassRateLimit.String())
	require.Equal(t, "server", ErrClassServer.String())
	require.Equal(t, "network", ErrClassNetwork.String())
	require.Equal(t, "permanent", ErrClassPermanent.String())
	require.Equal(t, "unknown", ErrClassUnknown.String())
}
