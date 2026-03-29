package auto

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/*.md.tpl
var templateFS embed.FS

// templateNames maps each UnitType to its template file name.
var templateNames = map[UnitType]string{
	UnitResearch:          "templates/research.md.tpl",
	UnitPlanSlice:         "templates/plan_slice.md.tpl",
	UnitExecuteTask:       "templates/execute_task.md.tpl",
	UnitSummarizeSlice:    "templates/summarize.md.tpl",
	UnitValidateMilestone: "templates/validate.md.tpl",
}

// PromptContext carries the runtime data injected into auto-mode prompt
// templates.
type PromptContext struct {
	// MilestoneID is the milestone being worked on.
	MilestoneID string
	// MilestoneTitle is the human-readable milestone title.
	MilestoneTitle string
	// SliceID is the slice being worked on (empty for milestone-scoped units).
	SliceID string
	// SliceTitle is the human-readable slice title.
	SliceTitle string
	// SliceGoal describes the slice objective.
	SliceGoal string
	// TaskID is set only for execute_task units.
	TaskID string
	// TaskTitle is the human-readable task title.
	TaskTitle string
	// TaskDescription is the full task description or steps block.
	TaskDescription string
	// PriorSummaries contains concatenated prior context for the agent.
	PriorSummaries string
	// WorkingDir is the absolute path to the working directory.
	WorkingDir string
}

// BuildPrompt selects the template for the given UnitType and renders it
// with the provided PromptContext.
func BuildPrompt(unitType UnitType, ctx PromptContext) (string, error) {
	name, ok := templateNames[unitType]
	if !ok {
		return "", fmt.Errorf("no template for unit type %q", unitType)
	}

	raw, err := templateFS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", name, err)
	}

	tmpl, err := template.New(unitType.String()).Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, ctx); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return sb.String(), nil
}

// String returns the string representation of a UnitType.
func (u UnitType) String() string {
	return string(u)
}

// InitPromptContext carries the runtime data injected into the init planning
// prompt template.
type InitPromptContext struct {
	// Vision is the user's high-level project vision string.
	Vision string
	// WorkingDir is the absolute path to the working directory.
	WorkingDir string
}

// BuildInitPrompt renders the init planning prompt template with the given
// context.
func BuildInitPrompt(ctx InitPromptContext) (string, error) {
	const name = "templates/init.md.tpl"

	raw, err := templateFS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", name, err)
	}

	tmpl, err := template.New("init").Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, ctx); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return sb.String(), nil
}
