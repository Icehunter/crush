package agent

import (
	"strings"

	"github.com/charmbracelet/crush/internal/config"
)

var planningKeywords = []string{
	"plan", "architect", "architecture", "design", "refactor",
	"restructure", "redesign", "strategy", "approach for",
	"how should i", "tradeoffs", "trade-offs", "think through",
	"big picture", "system design", "best way to",
	"what's the best", "how do i approach",
}

var backgroundKeywords = []string{
	"what is ", "what's ", "quick question", "summarize",
	"explain ", "tldr", "what does ", "how does ",
}

// classifyPromptTier returns the appropriate model tier for a prompt using
// keyword heuristics. Returns SelectedModelTypeMain by default.
func classifyPromptTier(prompt string) config.SelectedModelType {
	lower := strings.ToLower(strings.TrimSpace(prompt))
	for _, kw := range planningKeywords {
		if strings.Contains(lower, kw) {
			return config.SelectedModelTypePlanning
		}
	}
	if !strings.Contains(lower, "\n") && len(lower) < 80 {
		for _, kw := range backgroundKeywords {
			if strings.Contains(lower, kw) {
				return config.SelectedModelTypeBackground
			}
		}
	}
	return config.SelectedModelTypeMain
}
