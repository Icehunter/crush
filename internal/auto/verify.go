package auto

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/crush/internal/shell"
)

// DefaultMaxOutputLen is the maximum byte length of stdout/stderr captured in
// a VerificationResult before truncation. Keeps diagnostic prompts
// reasonable.
const DefaultMaxOutputLen = 4096

// VerificationResult captures the outcome of running a single verification
// command.
type VerificationResult struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
	Passed   bool          `json:"passed"`
}

// Verifier runs a set of verification commands against a working directory
// and returns per-command results.
type Verifier interface {
	RunVerification(ctx context.Context, workingDir string) ([]VerificationResult, error)
}

// ShellVerifier implements Verifier by executing commands via shell.Shell.
// It short-circuits on the first failure.
type ShellVerifier struct {
	Commands []string
	Logger   *slog.Logger
}

// NewShellVerifier creates a ShellVerifier for the given commands.
func NewShellVerifier(commands []string, logger *slog.Logger) *ShellVerifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &ShellVerifier{
		Commands: commands,
		Logger:   logger,
	}
}

// RunVerification executes each command sequentially. It stops on the first
// failure and returns all results collected so far.
func (v *ShellVerifier) RunVerification(ctx context.Context, workingDir string) ([]VerificationResult, error) {
	if len(v.Commands) == 0 {
		return nil, nil
	}

	var results []VerificationResult
	for _, cmd := range v.Commands {
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		s := shell.NewShell(&shell.Options{
			WorkingDir: workingDir,
		})

		start := time.Now()
		stdout, stderr, err := s.Exec(ctx, cmd)
		elapsed := time.Since(start)

		stdout = truncateOutput(stdout, DefaultMaxOutputLen)
		stderr = truncateOutput(stderr, DefaultMaxOutputLen)

		result := VerificationResult{
			Command:  cmd,
			Stdout:   stdout,
			Stderr:   stderr,
			Duration: elapsed,
		}

		if err != nil {
			// Shell returns an error for non-zero exit. Extract the
			// exit code if possible; default to 1.
			result.ExitCode = 1
			result.Passed = false
			v.Logger.Error("Verification command failed",
				"command", cmd,
				"exit_code", result.ExitCode,
				"stderr", truncateOutput(stderr, 512),
				"duration", elapsed,
			)
			results = append(results, result)
			return results, nil // Not a system error — just a failed check.
		}

		result.ExitCode = 0
		result.Passed = true
		v.Logger.Info("Verification command passed",
			"command", cmd,
			"duration", elapsed,
		)
		results = append(results, result)
	}
	return results, nil
}

// truncateOutput truncates s to the last maxLen bytes, prepending an
// ellipsis marker when content is trimmed.
func truncateOutput(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return fmt.Sprintf("…[truncated %d bytes]…\n%s", len(s)-maxLen, s[len(s)-maxLen:])
}

// FormatFailureDiagnostic builds a human-readable diagnostic string from
// failed verification results, suitable for inclusion in a retry prompt.
func FormatFailureDiagnostic(results []VerificationResult) string {
	var out string
	for _, r := range results {
		if r.Passed {
			continue
		}
		out += fmt.Sprintf("## Verification Failures\n\n")
		out += fmt.Sprintf("### ❌ `%s` (exit code %d)\n", r.Command, r.ExitCode)
		if r.Stderr != "" {
			out += fmt.Sprintf("```stderr\n%s\n```\n", r.Stderr)
		}
		if r.Stdout != "" {
			out += fmt.Sprintf("```stdout\n%s\n```\n", r.Stdout)
		}
		out += "\n"
	}
	return out
}
