package config

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
)

// ImportClaudeCode attempts to import a Claude Code subscription token as the
// Anthropic provider credentials. It checks CLAUDE_CODE_OAUTH_TOKEN
// (set by `claude setup-token`), and is a no-op when already configured or
// when no valid token is found.
func (s *ConfigStore) ImportClaudeCode() bool {
	if testing.Testing() {
		return false
	}

	if s.HasConfigField(ScopeGlobal, "providers.anthropic.api_key") || s.HasConfigField(ScopeGlobal, "providers.anthropic.oauth") {
		return false
	}

	envToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if envToken == "" {
		return false
	}

	if !strings.HasPrefix(envToken, "sk-ant-oat") {
		slog.Warn("CLAUDE_CODE_OAUTH_TOKEN does not look like a valid Claude OAuth token; skipping import")
		return false
	}

	slog.Info("Found CLAUDE_CODE_OAUTH_TOKEN. Authenticating with Claude subscription...")
	if err := s.SetConfigField(
		ScopeGlobal,
		"providers."+string(catwalk.InferenceProviderAnthropic)+".api_key", envToken,
	); err != nil {
		slog.Error("Unable to save Claude Code token from env", "error", err)
		return false
	}

	slog.Info("Claude Code (env token) successfully imported.")
	return true
}
