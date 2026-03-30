package auto

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// TierRating represents a user's feedback on model tier selection.
type TierRating struct {
	Timestamp time.Time `json:"timestamp"`
	UnitType  string    `json:"unit_type"`
	ModelTier string    `json:"model_tier"`
	Rating    string    `json:"rating"` // "over", "ok", "under"
}

// RoutingHistory tracks user feedback on model tier selections for
// adaptive routing. Persisted to a JSON file.
type RoutingHistory struct {
	mu      sync.Mutex
	path    string
	ratings []TierRating
}

// NewRoutingHistory creates a routing history backed by the given file.
func NewRoutingHistory(path string) *RoutingHistory {
	rh := &RoutingHistory{path: path}
	rh.load()
	return rh
}

// Rate records a user's feedback on the last unit's model tier.
func (rh *RoutingHistory) Rate(unitType, modelTier, rating string) {
	rh.mu.Lock()
	rh.ratings = append(rh.ratings, TierRating{
		Timestamp: time.Now(),
		UnitType:  unitType,
		ModelTier: modelTier,
		Rating:    rating,
	})
	rh.mu.Unlock()
	rh.save()
}

// Suggest returns the recommended tier adjustment for a unit type based
// on historical ratings. Returns "up", "down", or "" (no change).
func (rh *RoutingHistory) Suggest(unitType string) string {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	var overCount, underCount, total int
	for _, r := range rh.ratings {
		if r.UnitType != unitType {
			continue
		}
		total++
		switch r.Rating {
		case "over":
			overCount++
		case "under":
			underCount++
		}
	}

	if total < 3 {
		return "" // Not enough data.
	}

	overRatio := float64(overCount) / float64(total)
	underRatio := float64(underCount) / float64(total)

	if overRatio > 0.6 {
		return "down"
	}
	if underRatio > 0.6 {
		return "up"
	}
	return ""
}

// Summary returns a formatted summary of routing feedback.
func (rh *RoutingHistory) Summary() string {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	if len(rh.ratings) == 0 {
		return "No tier ratings recorded yet"
	}

	// Count by unit type.
	type stats struct{ over, ok, under int }
	byType := make(map[string]*stats)

	for _, r := range rh.ratings {
		s, exists := byType[r.UnitType]
		if !exists {
			s = &stats{}
			byType[r.UnitType] = s
		}
		switch r.Rating {
		case "over":
			s.over++
		case "ok":
			s.ok++
		case "under":
			s.under++
		}
	}

	result := "Tier rating history:\n"
	for ut, s := range byType {
		result += "  " + ut + ": "
		result += "over=" + itoa(s.over) + " ok=" + itoa(s.ok) + " under=" + itoa(s.under) + "\n"
	}
	return result
}

func (rh *RoutingHistory) load() {
	data, err := os.ReadFile(rh.path)
	if err != nil {
		return
	}
	var ratings []TierRating
	if err := json.Unmarshal(data, &ratings); err != nil {
		return
	}
	rh.mu.Lock()
	rh.ratings = ratings
	rh.mu.Unlock()
}

func (rh *RoutingHistory) save() {
	rh.mu.Lock()
	data, err := json.MarshalIndent(rh.ratings, "", "  ")
	rh.mu.Unlock()
	if err != nil {
		return
	}
	_ = os.WriteFile(rh.path, data, 0o644)
}

// itoa is a minimal int-to-string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
