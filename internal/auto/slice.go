package auto

import (
	"database/sql"

	"github.com/charmbracelet/crush/internal/db"
)

// Slice is the domain model wrapping the SQLC-generated db.Slice.
type Slice struct {
	ID          string `json:"id"`
	MilestoneID string `json:"milestone_id"`
	Title       string `json:"title"`
	Status      Status `json:"status"`
	Phase       Phase  `json:"phase"`
	SortOrder   int64  `json:"sort_order"`
	DependsOn   string `json:"depends_on"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// SliceFromDB converts a db.Slice into the typed domain model.
func SliceFromDB(s db.Slice) Slice {
	return Slice{
		ID:          s.ID,
		MilestoneID: s.MilestoneID,
		Title:       s.Title,
		Status:      Status(s.Status),
		Phase:       Phase(s.Phase),
		SortOrder:   s.SortOrder,
		DependsOn:   nullStringToString(s.DependsOn),
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

// ToDBCreate converts the domain model into parameters for db.CreateSlice.
func (s Slice) ToDBCreate() db.CreateSliceParams {
	return db.CreateSliceParams{
		ID:          s.ID,
		MilestoneID: s.MilestoneID,
		Title:       s.Title,
		Status:      string(s.Status),
		Phase:       string(s.Phase),
		SortOrder:   s.SortOrder,
		DependsOn:   stringToNullString(s.DependsOn),
	}
}

// nullStringToString converts a sql.NullString to a plain string.
// When Valid is false the result is an empty string.
func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// stringToNullString converts a plain string to sql.NullString.
// An empty string produces a NULL (Valid = false).
func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
