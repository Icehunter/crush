package auto

import "context"

// TokenQuerier returns cumulative token usage for a session. The engine
// uses this to detect context pressure before the model's context window
// is exhausted.
type TokenQuerier interface {
	GetTokenUsage(ctx context.Context, sessionID string) (promptTokens int64, completionTokens int64, err error)
}

// ContextMonitor compares cumulative session token usage against a
// configurable fraction of the model's context window. When usage
// exceeds the threshold the engine pauses and publishes an event.
type ContextMonitor struct {
	threshold     float64 // 0.0–1.0, fraction of contextWindow.
	contextWindow int64   // Model context window size in tokens.
	tokenQuerier  TokenQuerier
}

// NewContextMonitor creates a ContextMonitor. Returns nil (disabled) if
// contextWindow <= 0 or querier is nil.
func NewContextMonitor(threshold float64, contextWindow int64, querier TokenQuerier) *ContextMonitor {
	if contextWindow <= 0 || querier == nil {
		return nil
	}
	return &ContextMonitor{
		threshold:     threshold,
		contextWindow: contextWindow,
		tokenQuerier:  querier,
	}
}

// Check queries cumulative token usage for sessionID and returns true
// when (prompt + completion) / contextWindow >= threshold.
func (cm *ContextMonitor) Check(ctx context.Context, sessionID string) (bool, error) {
	prompt, completion, err := cm.tokenQuerier.GetTokenUsage(ctx, sessionID)
	if err != nil {
		return false, err
	}
	ratio := float64(prompt+completion) / float64(cm.contextWindow)
	return ratio >= cm.threshold, nil
}
