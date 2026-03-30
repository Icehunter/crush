package auto

import (
	"context"
	"fmt"
)

// MilestoneParker provides park/unpark operations on milestones.
// It uses the StateQuerier to check existence and the StatusAdvancer
// pattern (via direct DB calls) to toggle status.
type MilestoneParker struct {
	querier  StateQuerier
	advancer *DBStatusAdvancer
}

// NewMilestoneParker creates a MilestoneParker.
func NewMilestoneParker(querier StateQuerier, advancer *DBStatusAdvancer) *MilestoneParker {
	return &MilestoneParker{querier: querier, advancer: advancer}
}

// Park sets a milestone's status to "parked". DeriveState skips
// milestones with this status.
func (p *MilestoneParker) Park(ctx context.Context, milestoneID string) error {
	milestones, err := p.querier.ListMilestones(ctx)
	if err != nil {
		return fmt.Errorf("list milestones: %w", err)
	}

	found := false
	for _, m := range milestones {
		if m.ID == milestoneID {
			found = true
			if Status(m.Status) == StatusParked {
				return fmt.Errorf("milestone %s is already parked", milestoneID)
			}
			if Status(m.Status) == StatusCompleted {
				return fmt.Errorf("cannot park completed milestone %s", milestoneID)
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("milestone %s not found", milestoneID)
	}

	return p.advancer.SetMilestoneStatus(ctx, milestoneID, string(StatusParked))
}

// Unpark restores a parked milestone to active status.
func (p *MilestoneParker) Unpark(ctx context.Context, milestoneID string) error {
	milestones, err := p.querier.ListMilestones(ctx)
	if err != nil {
		return fmt.Errorf("list milestones: %w", err)
	}

	found := false
	for _, m := range milestones {
		if m.ID == milestoneID {
			found = true
			if Status(m.Status) != StatusParked {
				return fmt.Errorf("milestone %s is not parked (status: %s)", milestoneID, m.Status)
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("milestone %s not found", milestoneID)
	}

	return p.advancer.SetMilestoneStatus(ctx, milestoneID, string(StatusActive))
}
