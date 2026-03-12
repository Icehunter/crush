package agent

import (
	"testing"

	"github.com/charmbracelet/crush/internal/config"
)

func TestClassifyPromptTier(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		expected config.SelectedModelType
	}{
		// Planning tier
		{
			name:     "plan keyword",
			prompt:   "plan the implementation for auth",
			expected: config.SelectedModelTypePlanning,
		},
		{
			name:     "architect keyword",
			prompt:   "architect a distributed system",
			expected: config.SelectedModelTypePlanning,
		},
		{
			name:     "architecture keyword",
			prompt:   "what architecture should I use for microservices",
			expected: config.SelectedModelTypePlanning,
		},
		{
			name:     "refactor keyword",
			prompt:   "refactor the database layer",
			expected: config.SelectedModelTypePlanning,
		},
		{
			name:     "design keyword",
			prompt:   "design a caching strategy",
			expected: config.SelectedModelTypePlanning,
		},
		{
			name:     "system design keyword",
			prompt:   "system design for a job queue",
			expected: config.SelectedModelTypePlanning,
		},
		{
			name:     "tradeoffs keyword",
			prompt:   "what are the tradeoffs between SQL and NoSQL",
			expected: config.SelectedModelTypePlanning,
		},
		{
			name:     "how should i keyword",
			prompt:   "how should i structure the API",
			expected: config.SelectedModelTypePlanning,
		},
		{
			name:     "think through keyword",
			prompt:   "think through the implications of this change",
			expected: config.SelectedModelTypePlanning,
		},
		// Background tier
		{
			name:     "what is keyword short prompt",
			prompt:   "what is a closure",
			expected: config.SelectedModelTypeBackground,
		},
		{
			name:     "explain keyword short prompt",
			prompt:   "explain goroutines",
			expected: config.SelectedModelTypeBackground,
		},
		{
			name:     "how does keyword short prompt",
			prompt:   "how does garbage collection work",
			expected: config.SelectedModelTypeBackground,
		},
		{
			name:     "what does keyword short prompt",
			prompt:   "what does defer do",
			expected: config.SelectedModelTypeBackground,
		},
		{
			name:     "tldr keyword",
			prompt:   "tldr this function",
			expected: config.SelectedModelTypeBackground,
		},
		// Main tier (default)
		{
			name:     "regular coding task",
			prompt:   "fix the bug in utils.go line 42",
			expected: config.SelectedModelTypeMain,
		},
		{
			name:     "multi-line prompt falls through to main",
			prompt:   "explain what\nthis code does",
			expected: config.SelectedModelTypeMain,
		},
		{
			name:     "long background keyword prompt falls through to main",
			prompt:   "explain the entire history of programming languages and how they evolved over time from assembly",
			expected: config.SelectedModelTypeMain,
		},
		{
			name:     "add feature",
			prompt:   "add a retry mechanism to the HTTP client",
			expected: config.SelectedModelTypeMain,
		},
		{
			name:     "empty prompt",
			prompt:   "",
			expected: config.SelectedModelTypeMain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyPromptTier(tt.prompt)
			if got != tt.expected {
				t.Errorf("classifyPromptTier(%q) = %q, want %q", tt.prompt, got, tt.expected)
			}
		})
	}
}
