// Package gsd implements GSD integration for Crush, including preferences
// parsing and slash command routing.
package gsd

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/charmbracelet/crush/internal/config"
	"gopkg.in/yaml.v3"
)

// Preferences represents the GSD PREFERENCES.md YAML frontmatter schema.
type Preferences struct {
	Version                int                 `yaml:"version"`
	Mode                   string              `yaml:"mode,omitempty"`
	AlwaysUseSkills        []string            `yaml:"always_use_skills,omitempty"`
	PreferSkills           []string            `yaml:"prefer_skills,omitempty"`
	AvoidSkills            []string            `yaml:"avoid_skills,omitempty"`
	SkillRules             []string            `yaml:"skill_rules,omitempty"`
	CustomInstructions     []string            `yaml:"custom_instructions,omitempty"`
	Models                 map[string]string   `yaml:"models,omitempty"`
	SkillDiscovery         string              `yaml:"skill_discovery,omitempty"`
	SkillStalenessDays     int                 `yaml:"skill_staleness_days,omitempty"`
	AutoSupervisor         map[string]any      `yaml:"auto_supervisor,omitempty"`
	Git                    GitPreferences      `yaml:"git,omitempty"`
	UniqueMilestoneIDs     *bool               `yaml:"unique_milestone_ids,omitempty"`
	BudgetCeiling          string              `yaml:"budget_ceiling,omitempty"`
	BudgetEnforcement      string              `yaml:"budget_enforcement,omitempty"`
	ContextPauseThreshold  int                 `yaml:"context_pause_threshold,omitempty"`
	TokenProfile           string              `yaml:"token_profile,omitempty"`
	Phases                 PhasesPreferences   `yaml:"phases,omitempty"`
	DynamicRouting         DynamicRouting      `yaml:"dynamic_routing,omitempty"`
	AutoVisualize          *bool               `yaml:"auto_visualize,omitempty"`
	AutoReport             *bool               `yaml:"auto_report,omitempty"`
	Parallel               ParallelPreferences `yaml:"parallel,omitempty"`
	VerificationCommands   []string            `yaml:"verification_commands,omitempty"`
	VerificationAutoFix    *bool               `yaml:"verification_auto_fix,omitempty"`
	VerificationMaxRetries int                 `yaml:"verification_max_retries,omitempty"`
	Notifications          Notifications       `yaml:"notifications,omitempty"`
	Cmux                   CmuxPreferences     `yaml:"cmux,omitempty"`
	RemoteQuestions        RemoteQuestions     `yaml:"remote_questions,omitempty"`
	UATDispatch            string              `yaml:"uat_dispatch,omitempty"`
	PostUnitHooks          []string            `yaml:"post_unit_hooks,omitempty"`
	PreDispatchHooks       []string            `yaml:"pre_dispatch_hooks,omitempty"`
}

// GitPreferences holds git-related GSD preferences.
type GitPreferences struct {
	AutoPush           *bool  `yaml:"auto_push,omitempty"`
	PushBranches       string `yaml:"push_branches,omitempty"`
	Remote             string `yaml:"remote,omitempty"`
	Snapshots          *bool  `yaml:"snapshots,omitempty"`
	PreMergeCheck      string `yaml:"pre_merge_check,omitempty"`
	CommitType         string `yaml:"commit_type,omitempty"`
	MainBranch         string `yaml:"main_branch,omitempty"`
	MergeStrategy      string `yaml:"merge_strategy,omitempty"`
	Isolation          string `yaml:"isolation,omitempty"`
	ManageGitignore    *bool  `yaml:"manage_gitignore,omitempty"`
	WorktreePostCreate string `yaml:"worktree_post_create,omitempty"`
}

// PhasesPreferences controls which planning phases are skipped.
type PhasesPreferences struct {
	SkipResearch       *bool `yaml:"skip_research,omitempty"`
	SkipReassess       *bool `yaml:"skip_reassess,omitempty"`
	ReassessAfterSlice *bool `yaml:"reassess_after_slice,omitempty"`
	SkipSliceResearch  *bool `yaml:"skip_slice_research,omitempty"`
}

// DynamicRouting controls model routing behavior.
type DynamicRouting struct {
	Enabled           *bool             `yaml:"enabled,omitempty"`
	TierModels        map[string]string `yaml:"tier_models,omitempty"`
	EscalateOnFailure *bool             `yaml:"escalate_on_failure,omitempty"`
	BudgetPressure    string            `yaml:"budget_pressure,omitempty"`
	CrossProvider     *bool             `yaml:"cross_provider,omitempty"`
	Hooks             string            `yaml:"hooks,omitempty"`
}

// ParallelPreferences controls parallel execution.
type ParallelPreferences struct {
	Enabled       *bool  `yaml:"enabled,omitempty"`
	MaxWorkers    int    `yaml:"max_workers,omitempty"`
	BudgetCeiling string `yaml:"budget_ceiling,omitempty"`
	MergeStrategy string `yaml:"merge_strategy,omitempty"`
	AutoMerge     *bool  `yaml:"auto_merge,omitempty"`
}

// Notifications controls notification behavior.
type Notifications struct {
	Enabled     *bool `yaml:"enabled,omitempty"`
	OnComplete  *bool `yaml:"on_complete,omitempty"`
	OnError     *bool `yaml:"on_error,omitempty"`
	OnBudget    *bool `yaml:"on_budget,omitempty"`
	OnMilestone *bool `yaml:"on_milestone,omitempty"`
	OnAttention *bool `yaml:"on_attention,omitempty"`
}

// CmuxPreferences controls the cmux terminal multiplexer integration.
type CmuxPreferences struct {
	Enabled       *bool  `yaml:"enabled,omitempty"`
	Notifications *bool  `yaml:"notifications,omitempty"`
	Sidebar       string `yaml:"sidebar,omitempty"`
	Splits        string `yaml:"splits,omitempty"`
	Browser       string `yaml:"browser,omitempty"`
}

// RemoteQuestions controls remote question dispatch.
type RemoteQuestions struct {
	Channel             string `yaml:"channel,omitempty"`
	ChannelID           string `yaml:"channel_id,omitempty"`
	TimeoutMinutes      int    `yaml:"timeout_minutes,omitempty"`
	PollIntervalSeconds int    `yaml:"poll_interval_seconds,omitempty"`
}

// LoadPreferences reads global and project PREFERENCES.md files, parses their
// YAML frontmatter, and merges them with project values overriding global
// values. Missing files are not errors — they return zero-value fields.
// Malformed YAML returns an error.
func LoadPreferences(globalPath, projectPath string) (*Preferences, error) {
	global, err := loadPreferencesFile(globalPath)
	if err != nil {
		return nil, fmt.Errorf("global preferences: %w", err)
	}

	project, err := loadPreferencesFile(projectPath)
	if err != nil {
		return nil, fmt.Errorf("project preferences: %w", err)
	}

	merged := mergePreferences(global, project)
	return merged, nil
}

// loadPreferencesFile reads and parses a single PREFERENCES.md file. Returns
// a zero-value Preferences if the file does not exist.
func loadPreferencesFile(path string) (*Preferences, error) {
	if path == "" {
		return &Preferences{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Preferences{}, nil
		}
		return nil, err
	}

	frontmatter, _, err := splitFrontmatter(string(data))
	if err != nil {
		// File exists but has no frontmatter — treat as empty preferences.
		return &Preferences{}, nil
	}

	var prefs Preferences
	if err := yaml.Unmarshal([]byte(frontmatter), &prefs); err != nil {
		return nil, fmt.Errorf("parsing YAML in %s: %w", path, err)
	}

	return &prefs, nil
}

// splitFrontmatter extracts YAML frontmatter and body from markdown content.
// The frontmatter is the text between the opening and closing --- delimiters.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", "", errors.New("no YAML frontmatter found")
	}

	rest := strings.TrimPrefix(content, "---\n")
	before, after, ok := strings.Cut(rest, "\n---")
	if !ok {
		return "", "", errors.New("unclosed frontmatter")
	}

	return before, after, nil
}

// mergePreferences merges project preferences over global preferences.
// Non-zero project values override global values. For pointer fields (*bool),
// a non-nil project value overrides the global. For slices and maps, a
// non-empty project value replaces the global value entirely.
func mergePreferences(global, project *Preferences) *Preferences {
	if global == nil {
		global = &Preferences{}
	}
	if project == nil {
		return global
	}

	result := *global
	mergeStruct(reflect.ValueOf(&result).Elem(), reflect.ValueOf(project).Elem())
	return &result
}

// mergeStruct recursively merges non-zero src fields into dst.
func mergeStruct(dst, src reflect.Value) {
	for i := range dst.NumField() {
		dstField := dst.Field(i)
		srcField := src.Field(i)

		if !dstField.CanSet() {
			continue
		}

		switch srcField.Kind() {
		case reflect.Ptr:
			if !srcField.IsNil() {
				dstField.Set(srcField)
			}
		case reflect.Map:
			if srcField.Len() > 0 {
				dstField.Set(srcField)
			}
		case reflect.Slice:
			if srcField.Len() > 0 {
				dstField.Set(srcField)
			}
		case reflect.String:
			if srcField.String() != "" {
				dstField.Set(srcField)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if srcField.Int() != 0 {
				dstField.Set(srcField)
			}
		case reflect.Struct:
			mergeStruct(dstField, srcField)
		default:
			if !srcField.IsZero() {
				dstField.Set(srcField)
			}
		}
	}
}

// ApplyToAutoConfig maps relevant Preferences fields onto a config.AutoConfig.
// Only non-zero preference values override existing config values.
func (p *Preferences) ApplyToAutoConfig(ac *config.AutoConfig) {
	if ac == nil {
		return
	}

	if p.BudgetCeiling != "" {
		if v, err := parseBudget(p.BudgetCeiling); err == nil {
			ac.BudgetCeiling = v
		}
	}

	if len(p.VerificationCommands) > 0 {
		ac.VerificationCommands = p.VerificationCommands
	}

	if p.Git.Isolation != "" {
		ac.WorktreeMode = p.Git.Isolation
	}

	if p.BudgetEnforcement != "" {
		ac.BudgetEnforcement = p.BudgetEnforcement
	}

	if p.Phases.SkipResearch != nil && *p.Phases.SkipResearch {
		ac.SkipResearch = true
	}
	if p.Phases.SkipSliceResearch != nil && *p.Phases.SkipSliceResearch {
		ac.SkipSliceResearch = true
	}
}

// FormatPreferences returns a human-readable summary of the given preferences.
func FormatPreferences(p *Preferences) string {
	if p == nil {
		return "No preferences loaded"
	}
	var lines []string
	lines = append(lines, "GSD Preferences:")
	if p.Mode != "" {
		lines = append(lines, "  mode: "+p.Mode)
	}
	if p.BudgetCeiling != "" {
		lines = append(lines, "  budget_ceiling: "+p.BudgetCeiling)
	}
	if p.BudgetEnforcement != "" {
		lines = append(lines, "  budget_enforcement: "+p.BudgetEnforcement)
	}
	if p.Git.Isolation != "" {
		lines = append(lines, "  git.isolation: "+p.Git.Isolation)
	}
	if p.Git.AutoPush != nil {
		lines = append(lines, fmt.Sprintf("  git.auto_push: %v", *p.Git.AutoPush))
	}
	if p.Git.Remote != "" {
		lines = append(lines, "  git.remote: "+p.Git.Remote)
	}
	if p.Phases.SkipResearch != nil {
		lines = append(lines, fmt.Sprintf("  phases.skip_research: %v", *p.Phases.SkipResearch))
	}
	if p.Phases.SkipSliceResearch != nil {
		lines = append(lines, fmt.Sprintf("  phases.skip_slice_research: %v", *p.Phases.SkipSliceResearch))
	}
	if len(p.VerificationCommands) > 0 {
		lines = append(lines, "  verification_commands: "+strings.Join(p.VerificationCommands, ", "))
	}
	if len(lines) == 1 {
		lines = append(lines, "  (all defaults)")
	}
	return strings.Join(lines, "\n")
}

// SetPreferenceValue loads a PREFERENCES.md, sets a key=value, and writes it
// back. Supports dotted keys like "git.auto_push".
func SetPreferenceValue(path, key, value string) error {
	prefs, err := loadPreferencesFile(path)
	if err != nil {
		return err
	}

	if err := applyPreferenceValue(prefs, key, value); err != nil {
		return err
	}

	return writePreferencesFile(path, prefs)
}

// applyPreferenceValue sets a single field on a Preferences struct by key.
func applyPreferenceValue(p *Preferences, key, value string) error {
	boolVal := func(s string) (*bool, error) {
		switch strings.ToLower(s) {
		case "true", "1", "yes":
			v := true
			return &v, nil
		case "false", "0", "no":
			v := false
			return &v, nil
		default:
			return nil, fmt.Errorf("invalid bool value: %q", s)
		}
	}

	switch key {
	case "mode":
		p.Mode = value
	case "budget_ceiling":
		p.BudgetCeiling = value
	case "budget_enforcement":
		p.BudgetEnforcement = value
	case "git.auto_push", "auto_push":
		b, err := boolVal(value)
		if err != nil {
			return err
		}
		p.Git.AutoPush = b
	case "git.remote", "remote":
		p.Git.Remote = value
	case "git.isolation", "isolation":
		p.Git.Isolation = value
	case "git.snapshots", "snapshots":
		b, err := boolVal(value)
		if err != nil {
			return err
		}
		p.Git.Snapshots = b
	case "phases.skip_research", "skip_research":
		b, err := boolVal(value)
		if err != nil {
			return err
		}
		p.Phases.SkipResearch = b
	case "phases.skip_slice_research", "skip_slice_research":
		b, err := boolVal(value)
		if err != nil {
			return err
		}
		p.Phases.SkipSliceResearch = b
	default:
		return fmt.Errorf("unknown preference key: %q", key)
	}
	return nil
}

// writePreferencesFile writes a Preferences struct back to a PREFERENCES.md
// file with YAML frontmatter.
func writePreferencesFile(path string, p *Preferences) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal preferences: %w", err)
	}

	content := "---\n" + string(data) + "---\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

// parseBudget converts a budget string like "$5.00" or "5.00" to a float64.
func parseBudget(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	return strconv.ParseFloat(s, 64)
}
