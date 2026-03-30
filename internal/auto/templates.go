package auto

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/charmbracelet/crush/internal/db"
	"github.com/google/uuid"
)

// WorkflowTemplate defines a pre-configured milestone structure for
// common development workflows. Templates are Go structs, not YAML —
// they live in-memory and are used by /gsd start <template>.
type WorkflowTemplate struct {
	ID          string
	Name        string
	Description string
	Slices      []TemplateSlice
}

// TemplateSlice defines a slice within a workflow template.
type TemplateSlice struct {
	Title     string
	DependsOn string // Comma-separated slice indices (e.g. "S01").
	Tasks     []string
}

// TemplateCatalog holds all known workflow templates.
var TemplateCatalog = []WorkflowTemplate{
	{
		ID:          "bugfix",
		Name:        "Bug Fix",
		Description: "Triage, fix, verify, ship",
		Slices: []TemplateSlice{
			{Title: "Triage & Reproduce", Tasks: []string{"Reproduce the bug", "Identify root cause"}},
			{Title: "Fix", DependsOn: "S01", Tasks: []string{"Implement fix", "Add regression test"}},
			{Title: "Verify & Ship", DependsOn: "S02", Tasks: []string{"Run full test suite", "Update changelog"}},
		},
	},
	{
		ID:          "feature",
		Name:        "Feature",
		Description: "Scope, plan, implement, verify",
		Slices: []TemplateSlice{
			{Title: "Scope & Design", Tasks: []string{"Define requirements", "Design approach"}},
			{Title: "Implement Core", DependsOn: "S01", Tasks: []string{"Implement core functionality", "Write unit tests"}},
			{Title: "Polish & Integration", DependsOn: "S02", Tasks: []string{"Integration tests", "Documentation", "Code review prep"}},
		},
	},
	{
		ID:          "spike",
		Name:        "Spike",
		Description: "Scope, research, synthesize",
		Slices: []TemplateSlice{
			{Title: "Scope", Tasks: []string{"Define spike questions", "Identify research areas"}},
			{Title: "Research", DependsOn: "S01", Tasks: []string{"Investigate options", "Build proof of concept"}},
			{Title: "Synthesize", DependsOn: "S02", Tasks: []string{"Document findings", "Recommend approach"}},
		},
	},
	{
		ID:          "hotfix",
		Name:        "Hotfix",
		Description: "Minimal: fix and ship",
		Slices: []TemplateSlice{
			{Title: "Fix & Ship", Tasks: []string{"Implement fix", "Verify fix", "Deploy"}},
		},
	},
	{
		ID:          "refactor",
		Name:        "Refactor",
		Description: "Inventory, plan, migrate, verify",
		Slices: []TemplateSlice{
			{Title: "Inventory", Tasks: []string{"Identify code to refactor", "Assess impact"}},
			{Title: "Plan Migration", DependsOn: "S01", Tasks: []string{"Design new structure", "Plan migration steps"}},
			{Title: "Migrate", DependsOn: "S02", Tasks: []string{"Execute refactoring", "Update tests"}},
			{Title: "Verify", DependsOn: "S03", Tasks: []string{"Run full test suite", "Performance comparison"}},
		},
	},
}

// LookupTemplate finds a template by ID or name (case-insensitive).
// Returns nil if not found.
func LookupTemplate(nameOrID string) *WorkflowTemplate {
	for i := range TemplateCatalog {
		t := &TemplateCatalog[i]
		if equalsIgnoreCase(t.ID, nameOrID) || equalsIgnoreCase(t.Name, nameOrID) {
			return t
		}
	}
	return nil
}

// ListTemplateNames returns a formatted list of available templates.
func ListTemplateNames() []string {
	out := make([]string, len(TemplateCatalog))
	for i, t := range TemplateCatalog {
		out[i] = t.ID + " — " + t.Description
	}
	return out
}

// TemplateApplier creates DB records from a workflow template.
type TemplateApplier interface {
	ApplyTemplate(ctx context.Context, tmpl *WorkflowTemplate) (milestoneID string, err error)
}

// DBTemplateApplier implements TemplateApplier using sqlc Queries.
type DBTemplateApplier struct {
	q *db.Queries
}

// NewDBTemplateApplier creates a DBTemplateApplier.
func NewDBTemplateApplier(q *db.Queries) *DBTemplateApplier {
	return &DBTemplateApplier{q: q}
}

// ApplyTemplate creates a milestone, slices, and tasks from a template.
// Returns the generated milestone ID.
func (a *DBTemplateApplier) ApplyTemplate(ctx context.Context, tmpl *WorkflowTemplate) (string, error) {
	milestoneID := fmt.Sprintf("M-%s", uuid.New().String()[:8])

	_, err := a.q.CreateMilestone(ctx, db.CreateMilestoneParams{
		ID:     milestoneID,
		Title:  tmpl.Name,
		Status: string(StatusActive),
		Phase:  string(PhasePrePlanning),
	})
	if err != nil {
		return "", fmt.Errorf("create milestone: %w", err)
	}

	for i, s := range tmpl.Slices {
		sliceID := fmt.Sprintf("%s-S%02d", milestoneID, i+1)
		dependsOn := sql.NullString{}
		if s.DependsOn != "" {
			// Resolve template-relative depends (e.g. "S01") to actual IDs.
			dependsOn = sql.NullString{
				String: fmt.Sprintf("%s-%s", milestoneID, s.DependsOn),
				Valid:  true,
			}
		}

		_, err := a.q.CreateSlice(ctx, db.CreateSliceParams{
			ID:          sliceID,
			MilestoneID: milestoneID,
			Title:       s.Title,
			Status:      string(StatusPending),
			Phase:       string(PhasePrePlanning),
			SortOrder:   int64(i + 1),
			DependsOn:   dependsOn,
		})
		if err != nil {
			return "", fmt.Errorf("create slice %s: %w", sliceID, err)
		}

		for j, taskTitle := range s.Tasks {
			taskID := fmt.Sprintf("%s-T%02d", sliceID, j+1)
			_, err := a.q.CreateTask(ctx, db.CreateTaskParams{
				ID:          taskID,
				SliceID:     sliceID,
				MilestoneID: milestoneID,
				Title:       taskTitle,
				Status:      string(StatusPending),
				Phase:       string(PhasePrePlanning),
				SortOrder:   int64(j + 1),
			})
			if err != nil {
				return "", fmt.Errorf("create task %s: %w", taskID, err)
			}
		}
	}

	return milestoneID, nil
}

// SetTemplateApplier configures the template applier on an EngineController.
func (c *EngineController) SetTemplateApplier(applier TemplateApplier) {
	c.templateApplier = applier
}

func equalsIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
