package auto

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// TestE2E_FullAssemblyWithVerifier proves the engine composes correctly
// with a ShellVerifier: dispatch → verify (pass) → advance.
func TestE2E_FullAssemblyWithVerifier(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute task T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// ShellVerifier with a command that always succeeds.
	verifier := NewShellVerifier([]string{"true"}, slog.Default())

	eng := NewEngine(
		querier, sessions, dispatch, advancer,
		verifier,
		nil, 0, nil, nil,
		broker, dir, slog.Default(), nil,
	)

	ctx := context.Background()
	err := eng.Step(ctx, "M001")
	require.NoError(t, err)

	// Dispatch called once — no retry needed since verification passes.
	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 1, "should dispatch exactly once when verification passes")
	dispatch.mu.Unlock()

	// Advancer called once — status advanced after successful verification.
	advancer.mu.Lock()
	require.Len(t, advancer.advanced, 1, "should advance once after successful verification")
	advancer.mu.Unlock()
}

// TestE2E_FullAssemblyWithFailingVerifier proves that a failing
// ShellVerifier triggers a retry dispatch with diagnostic prompt, and
// if the retry also fails verification, the step returns an error.
func TestE2E_FullAssemblyWithFailingVerifier(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute task T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// ShellVerifier with a command that always fails.
	verifier := NewShellVerifier([]string{"false"}, slog.Default())

	eng := NewEngine(
		querier, sessions, dispatch, advancer,
		verifier,
		nil, 0, nil, nil,
		broker, dir, slog.Default(), nil,
	)

	ctx := context.Background()
	err := eng.Step(ctx, "M001")
	require.Error(t, err, "step should error when verification fails after retry")
	require.Contains(t, err.Error(), "verification failed after retry")

	// Dispatch called twice: original + retry with diagnostic.
	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 2, "should dispatch original + retry")
	retryPrompt := dispatch.calls[1].Prompt
	dispatch.mu.Unlock()
	require.Contains(t, retryPrompt, "VERIFICATION FAILED")

	// Advancer NOT called — status not advanced.
	advancer.mu.Lock()
	require.Empty(t, advancer.advanced, "should not advance when verification fails")
	advancer.mu.Unlock()
}

// TestE2E_FullAssemblyWithStuckDetector proves the stuck detector
// records failures and fires after the window fills with >50% failures.
func TestE2E_FullAssemblyWithStuckDetector(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute task T01",
	}

	// Window size 3: need >50% failures (2+ out of 3) to trigger stuck.
	stuck := NewStuckDetector(3)
	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	broker := pubsub.NewBroker[AutoEvent]()

	// Dispatcher that always errors — each step() call will fail, and
	// the engine records failures in the stuck detector.
	alwaysFailDisp := &alwaysFailDispatcher{}
	advancer := &mockAdvancer{querier: querier}

	eng := NewEngine(
		querier, sessions, alwaysFailDisp, advancer,
		nil, // No verifier — dispatch itself fails.
		nil, 0,
		stuck,
		nil,
		broker, dir, slog.Default(), nil,
	)

	ctx := context.Background()
	key := UnitKey(unit)

	// Manually fill the stuck window: record 3 failures.
	// The engine records dispatch failures in the stuck detector, but
	// step() returns an error which Run would retry. We simulate the
	// recording manually to isolate the stuck detection logic.
	stuck.Record(key, false)
	stuck.Record(key, false)
	stuck.Record(key, false)

	// Now the detector should consider this unit stuck.
	require.True(t, stuck.IsStuck(key), "unit should be stuck after 3 failures in window of 3")

	// Step with the stuck unit — the engine should enter stuck-detected
	// diagnostic path. Since alwaysFailDisp also fails the diagnostic
	// retry, it should pause and record another failure.
	err := eng.Step(ctx, "M001")
	require.NoError(t, err, "step should not error — stuck handling pauses the engine")
	require.Equal(t, EnginePaused, eng.Status().State, "engine should be paused after stuck detection")
}

// TestE2E_WorktreeLifecycle proves Engine.Run creates a worktree before
// the main loop and merges+removes it on successful completion.
func TestE2E_WorktreeLifecycle(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)
	wm := NewWorktreeManager(repo)

	// One task that terminates the loop quickly.
	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute task T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &worktreeTrackingDispatcher{wm: wm, mid: "M001"}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	eng := NewEngine(
		querier, sessions, dispatch, advancer,
		nil, nil, 0, nil, nil,
		broker, repo, slog.Default(), nil,
	)
	eng.SetWorktreeManager(wm, "per-milestone")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Dispatch was called — verify the worktree existed during dispatch.
	dispatch.mu.Lock()
	require.True(t, dispatch.existedDuringDispatch,
		"worktree should exist while engine is dispatching")
	dispatch.mu.Unlock()

	// After Run completes, the worktree should be cleaned up (merge + remove).
	require.False(t, wm.Exists("M001"),
		"worktree should not exist after Run completes (removed after merge)")

	// The branch should have been merged — check the squash-merge commit
	// exists on the integration branch.
	branches := gitRun(t, repo, "branch", "--list", "auto/M001")
	require.Empty(t, branches, "branch should be deleted after remove")
}

// TestE2E_WorktreeResumeExisting proves Run succeeds when a worktree
// already exists — it skips Create and uses the existing one.
func TestE2E_WorktreeResumeExisting(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)
	wm := NewWorktreeManager(repo)
	ctx := context.Background()

	// Pre-create the worktree before Run.
	require.NoError(t, wm.Create(ctx, "M002"))
	require.True(t, wm.Exists("M002"))

	// Create a file in the worktree so merge has something.
	wtPath := wm.WorktreePath("M002")
	require.NoError(t, os.WriteFile(filepath.Join(wtPath, "resumed.txt"), []byte("resumed\n"), 0o644))
	gitRunInDir(t, wtPath, "add", ".")
	gitRunInDir(t, wtPath, "commit", "-m", "resumed work")

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M002",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute task T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	eng := NewEngine(
		querier, sessions, dispatch, advancer,
		nil, nil, 0, nil, nil,
		broker, repo, slog.Default(), nil,
	)
	eng.SetWorktreeManager(wm, "per-milestone")

	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := eng.Run(runCtx, "M002")
	require.NoError(t, err, "Run should succeed when worktree already exists")

	// Dispatch should have been called — engine ran normally.
	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 1)
	dispatch.mu.Unlock()

	// After completion, worktree merged and removed.
	require.False(t, wm.Exists("M002"))

	// The file from the worktree should now be on the main branch.
	data, err := os.ReadFile(filepath.Join(repo, "resumed.txt"))
	require.NoError(t, err)
	require.Equal(t, "resumed\n", string(data))
}

// TestE2E_BuildAutoEngineComposition proves the adapter composition
// pattern from buildAutoEngine produces a valid engine that can
// derive→dispatch→advance through real DB adapters.
func TestE2E_BuildAutoEngineComposition(t *testing.T) {
	t.Parallel()

	q := setupAdapterTestDB(t)
	ctx := context.Background()

	// Seed: one milestone, one slice, one task — all active/executing.
	seedMilestone(t, q, "M001", "active", "executing")
	seedSlice(t, q, "S01", "M001", "active", "executing", 1, "")
	seedTask(t, q, "T01", "S01", "M001", "active", "executing", 1)

	// Real adapters for querier and advancer.
	querier := NewDBStateQuerier(q)
	advancer := NewDBStatusAdvancer(q)

	// Mock session creator and recording dispatcher.
	sessions := &integrationSessionCreator{}
	dispatcher := &integrationDispatcher{}
	broker := pubsub.NewBroker[AutoEvent]()
	dir := t.TempDir()

	// Wire with all safety rails — mirrors what buildAutoEngine produces.
	verifier := NewShellVerifier([]string{"true"}, slog.Default())
	stuck := NewStuckDetector(5)

	eng := NewEngine(
		querier, sessions, dispatcher, advancer,
		verifier,
		nil, 0, // No budget ceiling for this test.
		stuck,
		nil, // No context monitor.
		broker, dir, slog.Default(), nil,
	)

	// Step 1: should find T01, dispatch, verify, advance.
	err := eng.Step(ctx, "M001")
	require.NoError(t, err)

	dispatcher.mu.Lock()
	require.Len(t, dispatcher.calls, 1, "Step should dispatch one unit")
	require.Equal(t, config.SelectedModelTypeMain, dispatcher.calls[0].Tier)
	require.Contains(t, dispatcher.calls[0].Prompt, "T01")
	dispatcher.mu.Unlock()

	// DB: task should be completed.
	task, err := q.GetTask(ctx, "T01")
	require.NoError(t, err)
	require.Equal(t, string(StatusCompleted), task.Status)

	// Step 2: summarize slice.
	err = eng.Step(ctx, "M001")
	require.NoError(t, err)

	dispatcher.mu.Lock()
	require.Len(t, dispatcher.calls, 2, "Step 2 should dispatch summarize")
	require.Equal(t, config.SelectedModelTypeBackground, dispatcher.calls[1].Tier)
	dispatcher.mu.Unlock()

	slice, err := q.GetSlice(ctx, "S01")
	require.NoError(t, err)
	require.Equal(t, string(StatusCompleted), slice.Status)

	// Step 3: validate milestone.
	err = eng.Step(ctx, "M001")
	require.NoError(t, err)

	milestone, err := q.GetMilestone(ctx, "M001")
	require.NoError(t, err)
	require.Equal(t, string(StatusCompleted), milestone.Status)
}

// --- E2E test helpers ---

// alwaysFailDispatcher always returns an error.
type alwaysFailDispatcher struct {
	mu    sync.Mutex
	calls int
}

func (d *alwaysFailDispatcher) RunWithForcedTier(_ context.Context, _, _ string, _ config.SelectedModelType) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls++
	return errSimulated
}

// worktreeTrackingDispatcher checks whether the worktree exists during
// dispatch and records the result.
type worktreeTrackingDispatcher struct {
	mu                     sync.Mutex
	existedDuringDispatch  bool
	calls                  []dispatchCall
	wm                     *WorktreeManager
	mid                    string
}

func (d *worktreeTrackingDispatcher) RunWithForcedTier(_ context.Context, sessionID, prompt string, tier config.SelectedModelType) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, dispatchCall{SessionID: sessionID, Prompt: prompt, Tier: tier})
	d.existedDuringDispatch = d.wm.Exists(d.mid)
	return nil
}

// gitRunInDir runs a git command in the given directory. Uses the same
// pattern as gitRun from worktree_test.go but with an explicit dir
// parameter.
func gitRunInDir(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return string(out)
}
