package auto

import (
	"context"
	"fmt"
	"strings"
)

// DeriveQueue returns a list of formatted unit descriptions representing the
// pending dispatch order for a milestone. It walks all slices and tasks,
// respecting dependency constraints, to produce a human-readable queue.
func DeriveQueue(ctx context.Context, q StateQuerier, milestoneID string) ([]string, error) {
	slices, err := q.ListSlicesByMilestone(ctx, milestoneID)
	if err != nil {
		return nil, fmt.Errorf("list slices: %w", err)
	}

	completed := make(map[string]bool, len(slices))
	for _, s := range slices {
		if Status(s.Status) == StatusCompleted {
			completed[s.ID] = true
		}
	}

	var queue []string

	for _, s := range slices {
		if Status(s.Status) == StatusCompleted {
			continue
		}

		blocked := !depsMetFor(s.DependsOn, completed)
		phase := Phase(s.Phase)

		switch phase {
		case PhasePrePlanning, PhaseResearching:
			entry := fmt.Sprintf("research %s — %s", s.ID, s.Title)
			if blocked {
				entry += " [blocked]"
			}
			queue = append(queue, entry)
			// After research, planning and execution will follow.
			queue = append(queue, fmt.Sprintf("  plan %s", s.ID))
			queue = append(queue, fmt.Sprintf("  execute tasks for %s", s.ID))
			queue = append(queue, fmt.Sprintf("  summarize %s", s.ID))

		case PhasePlanning:
			entry := fmt.Sprintf("plan %s — %s", s.ID, s.Title)
			if blocked {
				entry += " [blocked]"
			}
			queue = append(queue, entry)
			queue = append(queue, fmt.Sprintf("  execute tasks for %s", s.ID))
			queue = append(queue, fmt.Sprintf("  summarize %s", s.ID))

		case PhaseExecuting:
			tasks, terr := q.ListTasksBySlice(ctx, s.ID)
			if terr != nil {
				return nil, fmt.Errorf("list tasks for slice %s: %w", s.ID, terr)
			}
			for _, t := range tasks {
				if Status(t.Status) == StatusCompleted {
					continue
				}
				entry := fmt.Sprintf("execute %s — %s", t.ID, t.Title)
				if blocked {
					entry += " [blocked]"
				}
				queue = append(queue, entry)
			}
			queue = append(queue, fmt.Sprintf("summarize %s — %s", s.ID, s.Title))

		case PhaseSummarizing:
			queue = append(queue, fmt.Sprintf("summarize %s — %s", s.ID, s.Title))

		default:
			queue = append(queue, fmt.Sprintf("%s %s — %s", strings.ToLower(string(phase)), s.ID, s.Title))
		}
	}

	// If all non-completed slices have been listed, add validation.
	if len(queue) > 0 {
		queue = append(queue, fmt.Sprintf("validate milestone %s", milestoneID))
	}

	return queue, nil
}
