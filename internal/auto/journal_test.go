package auto

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJournal_RecordAndRead(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "journal")
	j := NewJournal(dir, "flow-001")

	now := time.Now()
	j.Record(JournalEntry{
		Timestamp:   now,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		UnitType:    string(UnitExecuteTask),
		UnitTitle:   "Login feature",
		Success:     true,
		DurationMs:  1500,
		Cost:        0.0042,
		ModelTier:   "main",
		SessionID:   "sess-123",
	})
	j.Record(JournalEntry{
		Timestamp:   now,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T02",
		UnitType:    string(UnitExecuteTask),
		UnitTitle:   "Logout feature",
		Success:     false,
		ErrorMsg:    "build failed",
		ErrorClass:  "unknown",
		DurationMs:  800,
	})

	entries, err := ReadEntries(dir, now)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	require.Equal(t, "flow-001", entries[0].FlowID)
	require.Equal(t, "Login feature", entries[0].UnitTitle)
	require.True(t, entries[0].Success)

	require.Equal(t, "Logout feature", entries[1].UnitTitle)
	require.False(t, entries[1].Success)
	require.Equal(t, "build failed", entries[1].ErrorMsg)
}

func TestJournal_ReadNonExistent(t *testing.T) {
	t.Parallel()
	entries, err := ReadEntries(t.TempDir(), time.Now())
	require.NoError(t, err)
	require.Nil(t, entries)
}

func TestReadRecentEntries(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "journal")
	j := NewJournal(dir, "flow-002")

	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	j.Record(JournalEntry{
		Timestamp:   yesterday,
		MilestoneID: "M001",
		UnitType:    string(UnitResearch),
		UnitTitle:   "Yesterday's work",
		Success:     true,
	})
	j.Record(JournalEntry{
		Timestamp:   today,
		MilestoneID: "M001",
		UnitType:    string(UnitPlanSlice),
		UnitTitle:   "Today's work",
		Success:     true,
	})

	entries, err := ReadRecentEntries(dir, 2)
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestBuildPriorSummaries_Empty(t *testing.T) {
	t.Parallel()
	require.Equal(t, "", BuildPriorSummaries(nil))
	require.Equal(t, "", BuildPriorSummaries([]JournalEntry{}))
}

func TestBuildPriorSummaries_OnlyFailures(t *testing.T) {
	t.Parallel()
	entries := []JournalEntry{
		{UnitType: "execute_task", UnitTitle: "Failed", Success: false},
	}
	require.Equal(t, "", BuildPriorSummaries(entries))
}

func TestBuildPriorSummaries_WithSuccesses(t *testing.T) {
	t.Parallel()
	entries := []JournalEntry{
		{
			UnitType:   string(UnitResearch),
			UnitTitle:  "Auth research",
			Success:    true,
			Cost:       0.005,
			DurationMs: 2000,
			SliceID:    "S01",
		},
		{
			UnitType:   string(UnitExecuteTask),
			UnitTitle:  "Login task",
			Success:    true,
			Cost:       0.01,
			DurationMs: 5000,
			SliceID:    "S01",
			TaskID:     "T01",
		},
	}

	result := BuildPriorSummaries(entries)
	require.Contains(t, result, "Previously completed")
	require.Contains(t, result, "Auth research")
	require.Contains(t, result, "Login task")
}
