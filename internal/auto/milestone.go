package auto

import (
	"github.com/charmbracelet/crush/internal/db"
)

// Milestone is the domain model wrapping the SQLC-generated db.Milestone.
type Milestone struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    Status `json:"status"`
	Phase     Phase  `json:"phase"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// MilestoneFromDB converts a db.Milestone into the typed domain model.
func MilestoneFromDB(m db.Milestone) Milestone {
	return Milestone{
		ID:        m.ID,
		Title:     m.Title,
		Status:    Status(m.Status),
		Phase:     Phase(m.Phase),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// ToDBCreate converts the domain model into parameters for db.CreateMilestone.
func (m Milestone) ToDBCreate() db.CreateMilestoneParams {
	return db.CreateMilestoneParams{
		ID:     m.ID,
		Title:  m.Title,
		Status: string(m.Status),
		Phase:  string(m.Phase),
	}
}
