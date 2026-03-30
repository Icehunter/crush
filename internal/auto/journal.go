package auto

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// JournalEntry records the outcome of a single unit dispatch.
type JournalEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	FlowID      string    `json:"flow_id"`
	MilestoneID string    `json:"milestone_id"`
	SliceID     string    `json:"slice_id,omitempty"`
	TaskID      string    `json:"task_id,omitempty"`
	UnitType    string    `json:"unit_type"`
	UnitTitle   string    `json:"unit_title"`
	Success     bool      `json:"success"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
	ErrorClass  string    `json:"error_class,omitempty"`
	DurationMs  int64     `json:"duration_ms"`
	Cost        float64   `json:"cost,omitempty"`
	ModelTier   string    `json:"model_tier,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
}

// Journal writes structured entries to daily JSONL files under a journal
// directory. Writes use O_APPEND|O_CREATE for safe concurrent appends.
// All errors are silent — journaling must never block the engine.
type Journal struct {
	dir    string
	flowID string
}

// NewJournal creates a journal that writes to dir/YYYY-MM-DD.jsonl files.
// flowID identifies this particular auto-mode run.
func NewJournal(dir, flowID string) *Journal {
	return &Journal{dir: dir, flowID: flowID}
}

// Record appends an entry to the daily journal file.
func (j *Journal) Record(entry JournalEntry) {
	entry.FlowID = j.flowID
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')

	filename := entry.Timestamp.Format("2006-01-02") + ".jsonl"
	path := filepath.Join(j.dir, filename)

	if mkErr := os.MkdirAll(j.dir, 0o755); mkErr != nil {
		return
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(data)
}

// ReadEntries reads all entries from a specific date's journal file.
// Returns nil (not error) if the file doesn't exist.
func ReadEntries(dir string, date time.Time) ([]JournalEntry, error) {
	filename := date.Format("2006-01-02") + ".jsonl"
	path := filepath.Join(dir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read journal %s: %w", path, err)
	}

	var entries []JournalEntry
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var e JournalEntry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // Skip malformed lines.
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ReadRecentEntries reads journal entries from the last N days.
func ReadRecentEntries(dir string, days int) ([]JournalEntry, error) {
	var all []JournalEntry
	now := time.Now()
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)
		entries, err := ReadEntries(dir, date)
		if err != nil {
			return nil, err
		}
		all = append(all, entries...)
	}
	return all, nil
}

// BuildPriorSummaries formats completed slice/task summaries from recent
// journal entries into a string suitable for injection into prompt context.
func BuildPriorSummaries(entries []JournalEntry) string {
	if len(entries) == 0 {
		return ""
	}

	// Collect unique completed units (most recent wins).
	type key struct{ unitType, sliceID, taskID string }
	seen := make(map[key]JournalEntry)
	for _, e := range entries {
		if !e.Success {
			continue
		}
		k := key{e.UnitType, e.SliceID, e.TaskID}
		// Later entries overwrite earlier ones.
		seen[k] = e
	}

	if len(seen) == 0 {
		return ""
	}

	var lines []string
	for _, e := range seen {
		line := fmt.Sprintf("- [%s] %s (cost: $%.4f, %dms)",
			e.UnitType, e.UnitTitle, e.Cost, e.DurationMs)
		lines = append(lines, line)
	}

	result := "Previously completed work in this milestone:\n"
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}

// splitLines splits data on newline boundaries.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
