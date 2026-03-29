package auto

import (
	"context"
	"fmt"
	"log/slog"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent"
	agentnotify "github.com/charmbracelet/crush/internal/agent/notify"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/charmbracelet/crush/internal/session"
)

// InitConfig holds the dependencies required by RunInit.
type InitConfig struct {
	// Vision is the user's high-level project description.
	Vision string
	// WorkingDir is the absolute path to the working directory.
	WorkingDir string
	// Queries is the database accessor for persisting milestones, slices,
	// and tasks.
	Queries *db.Queries
	// Sessions is the session service for creating agent sessions.
	Sessions session.Service
	// Messages is the message service required by the SessionAgent.
	Messages message.Service
	// Model is the LLM model to use for planning.
	Model agent.Model
	// Logger is the structured logger.
	Logger *slog.Logger
}

// RunInit orchestrates the interactive planning flow. It constructs a
// SessionAgent with the three planning tools and the init prompt, then
// dispatches the user's vision as the user message.
func RunInit(ctx context.Context, cfg InitConfig) error {
	if cfg.Vision == "" {
		return fmt.Errorf("vision is required")
	}
	if cfg.Queries == nil {
		return fmt.Errorf("queries is required")
	}
	if cfg.Sessions == nil {
		return fmt.Errorf("sessions service is required")
	}
	if cfg.Messages == nil {
		return fmt.Errorf("messages service is required")
	}
	if cfg.Model.Model == nil {
		return fmt.Errorf("model is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Build the init system prompt.
	systemPrompt, err := BuildInitPrompt(InitPromptContext{
		Vision:     cfg.Vision,
		WorkingDir: cfg.WorkingDir,
	})
	if err != nil {
		return fmt.Errorf("build init prompt: %w", err)
	}

	// Create a session for this init run.
	sess, err := cfg.Sessions.Create(ctx, "auto-init: "+cfg.Vision)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	logger.Info("Created init session", "session_id", sess.ID)

	// Build the three planning tools.
	tools := []fantasy.AgentTool{
		NewCreateMilestoneTool(cfg.Queries),
		NewCreateSliceTool(cfg.Queries),
		NewCreateTaskTool(cfg.Queries),
	}

	// Construct a SessionAgent with only the planning tools.
	notifyBroker := pubsub.NewBroker[agentnotify.Notification]()
	defer notifyBroker.Shutdown()

	sa := agent.NewSessionAgent(agent.SessionAgentOptions{
		MainModel:    cfg.Model,
		SystemPrompt: systemPrompt,
		Tools:        tools,
		Sessions:     cfg.Sessions,
		Messages:     cfg.Messages,
		Notify:       notifyBroker,
		IsSubAgent:   true,
	})

	// Dispatch the vision as a non-interactive prompt.
	logger.Info("Dispatching init planning", "vision", cfg.Vision)
	_, err = sa.Run(ctx, agent.SessionAgentCall{
		SessionID:      sess.ID,
		Prompt:         cfg.Vision,
		NonInteractive: true,
	})
	if err != nil {
		return fmt.Errorf("init planning failed: %w", err)
	}

	logger.Info("Init planning completed", "session_id", sess.ID)
	return nil
}
