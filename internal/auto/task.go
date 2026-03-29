package auto

import (
	"github.com/charmbracelet/crush/internal/db"
)

// Task is the domain model wrapping the SQLC-generated db.Task.
type Task struct {
	ID          string `json:"id"`
	SliceID     string `json:"slice_id"`
	MilestoneID string `json:"milestone_id"`
	Title       string `json:"title"`
	Status      Status `json:"status"`
	Phase       Phase  `json:"phase"`
	SortOrder   int64  `json:"sort_order"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// TaskFromDB converts a db.Task into the typed domain model.
func TaskFromDB(t db.Task) Task {
	return Task{
		ID:          t.ID,
		SliceID:     t.SliceID,
		MilestoneID: t.MilestoneID,
		Title:       t.Title,
		Status:      Status(t.Status),
		Phase:       Phase(t.Phase),
		SortOrder:   t.SortOrder,
		Description: nullStringToString(t.Description),
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// ToDBCreate converts the domain model into parameters for db.CreateTask.
func (t Task) ToDBCreate() db.CreateTaskParams {
	return db.CreateTaskParams{
		ID:          t.ID,
		SliceID:     t.SliceID,
		MilestoneID: t.MilestoneID,
		Title:       t.Title,
		Status:      string(t.Status),
		Phase:       string(t.Phase),
		SortOrder:   t.SortOrder,
		Description: stringToNullString(t.Description),
	}
}
