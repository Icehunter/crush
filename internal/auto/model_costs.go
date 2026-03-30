package auto

// ModelCost represents per-token pricing for a model.
type ModelCost struct {
	PromptPerMToken     float64 // Cost per million prompt tokens.
	CompletionPerMToken float64 // Cost per million completion tokens.
}

// EstimateCost calculates the cost given token counts.
func (c ModelCost) EstimateCost(promptTokens, completionTokens int64) float64 {
	prompt := float64(promptTokens) / 1_000_000 * c.PromptPerMToken
	completion := float64(completionTokens) / 1_000_000 * c.CompletionPerMToken
	return prompt + completion
}

// Known model costs (as of early 2026). These are approximate and should
// be updated when pricing changes. The engine uses these for cost estimation
// in metrics; actual billing comes from the provider.
var knownModelCosts = map[string]ModelCost{
	// Claude 4.x family
	"claude-opus-4-6":          {PromptPerMToken: 15.0, CompletionPerMToken: 75.0},
	"claude-sonnet-4-6":        {PromptPerMToken: 3.0, CompletionPerMToken: 15.0},
	"claude-haiku-4-5-20251001": {PromptPerMToken: 0.80, CompletionPerMToken: 4.0},

	// Claude 3.5 family (legacy)
	"claude-3-5-sonnet-20241022": {PromptPerMToken: 3.0, CompletionPerMToken: 15.0},
	"claude-3-5-haiku-20241022":  {PromptPerMToken: 0.80, CompletionPerMToken: 4.0},
}

// LookupModelCost returns the cost structure for a model ID.
// Returns a zero-value ModelCost if the model is unknown.
func LookupModelCost(modelID string) ModelCost {
	if c, ok := knownModelCosts[modelID]; ok {
		return c
	}
	return ModelCost{}
}
