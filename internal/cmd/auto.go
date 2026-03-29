package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	appPkg "github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/auto"
	"github.com/spf13/cobra"
)

var autoStatusJSON bool

var autoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Manage auto-mode execution",
	Long:  "Manage auto-mode execution for milestones. Use subcommands to start, pause, stop, or check status.",
}

var autoStartCmd = &cobra.Command{
	Use:   "start <milestone-id>",
	Short: "Start auto-mode for a milestone",
	Long:  "Start the auto-mode engine loop for the given milestone. Runs until all work is done, interrupted, or a budget ceiling is reached.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoStart,
}

var autoPauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause auto-mode execution",
	Long:  "Pause is only available in TUI mode (ctrl+a) or by sending SIGINT to the running process.",
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Println("Pause is only available in TUI mode (ctrl+a) or send SIGINT to the running process.")
		return nil
	},
}

var autoStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop auto-mode execution",
	Long:  "Stop is only available in TUI mode (ctrl+a) or by sending SIGINT to the running process.",
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Println("Stop is only available in TUI mode (ctrl+a) or send SIGINT to the running process.")
		return nil
	},
}

var autoStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check auto-mode status",
	Long:  "Check whether auto-mode is currently running by inspecting the lock file.",
	RunE:  runAutoStatus,
}

var nextCmd = &cobra.Command{
	Use:   "next <milestone-id>",
	Short: "Run exactly one auto-mode unit",
	Long:  "Run exactly one auto-mode unit (derive → dispatch → advance) for the given milestone, then exit.",
	Args:  cobra.ExactArgs(1),
	RunE:  runNext,
}

func init() {
	autoStatusCmd.Flags().BoolVar(&autoStatusJSON, "json", false, "Output in JSON format")
	autoCmd.AddCommand(autoStartCmd, autoPauseCmd, autoStopCmd, autoStatusCmd)
	rootCmd.AddCommand(autoCmd, nextCmd)
}

func runAutoStart(cmd *cobra.Command, args []string) error {
	milestoneID := args[0]

	app, err := setupApp(cmd)
	if err != nil {
		return err
	}
	defer app.Shutdown()

	engine, err := buildAutoEngine(app)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	slog.Info("Starting auto-mode", "milestone", milestoneID)
	if err := engine.Run(ctx, milestoneID); err != nil {
		return fmt.Errorf("auto-mode failed: %w", err)
	}

	fmt.Println("Auto-mode completed.")
	return nil
}

func runAutoStatus(cmd *cobra.Command, _ []string) error {
	app, err := setupApp(cmd)
	if err != nil {
		return err
	}
	defer app.Shutdown()

	cfg := app.Config()
	lock := auto.NewLockFile(cfg.Options.DataDirectory)

	// Read the lock file to determine status.
	data, readErr := os.ReadFile(lock.Path())

	type statusOutput struct {
		Running   bool   `json:"running"`
		PID       int    `json:"pid,omitempty"`
		StartedAt string `json:"started_at,omitempty"`
	}

	if readErr != nil {
		// No lock file — not running.
		if autoStatusJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(statusOutput{Running: false})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Auto-mode is not running.")
		return nil
	}

	var payload struct {
		PID       int    `json:"pid"`
		StartedAt string `json:"started_at"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		// Lock file unreadable — treat as not running.
		if autoStatusJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(statusOutput{Running: false})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Auto-mode is not running (lock file unreadable).")
		return nil
	}

	running := isProcessAlive(payload.PID)

	if autoStatusJSON {
		out := statusOutput{
			Running:   running,
			PID:       payload.PID,
			StartedAt: payload.StartedAt,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetEscapeHTML(false)
		return enc.Encode(out)
	}

	if running {
		fmt.Fprintf(cmd.OutOrStdout(), "Auto-mode is running (pid %d, started %s).\n", payload.PID, payload.StartedAt)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Auto-mode is not running (stale lock file).")
	}
	return nil
}

func runNext(cmd *cobra.Command, args []string) error {
	milestoneID := args[0]

	app, err := setupApp(cmd)
	if err != nil {
		return err
	}
	defer app.Shutdown()

	engine, err := buildAutoEngine(app)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	slog.Info("Running next unit", "milestone", milestoneID)
	if err := engine.Step(ctx, milestoneID); err != nil {
		return fmt.Errorf("next step failed: %w", err)
	}

	fmt.Println("Step completed.")
	return nil
}

// buildAutoEngine constructs the auto-mode engine from the given App's
// dependencies. This is the shared helper used by all auto/next commands.
func buildAutoEngine(app *appPkg.App) (*auto.Engine, error) {
	if app.Queries == nil {
		return nil, fmt.Errorf("database not available")
	}
	if app.AgentCoordinator == nil {
		return nil, fmt.Errorf("agent coordinator not available")
	}

	cfg := app.Config()
	querier := auto.NewDBStateQuerier(app.Queries)
	sessions := auto.NewSessionServiceCreator(app.Sessions)
	dispatcher := auto.NewCoordinatorDispatcher(app.AgentCoordinator)
	advancer := auto.NewDBStatusAdvancer(app.Queries)
	budgetChecker := auto.NewDBBudgetChecker(app.Queries)
	dataDir := cfg.Options.DataDirectory

	var budgetCeiling float64
	if cfg.Auto != nil {
		budgetCeiling = cfg.Auto.BudgetCeiling
	}

	// Wire optional safety rails from config.
	var verifier auto.Verifier
	if cfg.Auto != nil && len(cfg.Auto.VerificationCommands) > 0 {
		verifier = auto.NewShellVerifier(cfg.Auto.VerificationCommands, slog.Default())
	}

	var stuckDetector *auto.StuckDetector
	if cfg.Auto != nil && cfg.Auto.StuckThreshold > 0 {
		stuckDetector = auto.NewStuckDetector(cfg.Auto.StuckThreshold)
	}

	engine := auto.NewEngine(
		querier,
		sessions,
		dispatcher,
		advancer,
		verifier,
		budgetChecker,
		budgetCeiling,
		stuckDetector,
		nil, // contextMonitor
		appPkg.AutoBroker(),
		dataDir,
		slog.Default(),
		querier, // snapshotQuerier
	)

	// Wire worktree manager if configured.
	if cfg.Auto != nil && cfg.Auto.WorktreeMode == "per-milestone" {
		projectRoot, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory for worktree manager: %w", err)
		}
		wm := auto.NewWorktreeManager(projectRoot)
		engine.SetWorktreeManager(wm, cfg.Auto.WorktreeMode)
	}

	return engine, nil
}

// isProcessAlive checks whether a process with the given PID is running.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 tests for existence without affecting the process.
	return proc.Signal(syscall.Signal(0)) == nil
}
