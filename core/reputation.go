package core

import (
	"math"
)

// Reputation represents a user's reputation score and history.
type Reputation struct {
	Score              float64 `json:"score"`
	Contributions      uint64  `json:"contributions"`
	CheatAttempts      uint64  `json:"cheatAttempts"`
	InvalidMFAAttempts uint64  `json:"invalidMFAAttempts"`
}

// NewReputation initializes a new Reputation with a default score.
func NewReputation() *Reputation {
	return &Reputation{
		Score:              1.0,
		Contributions:      0,
		CheatAttempts:      0,
		InvalidMFAAttempts: 0,
	}
}

// UpdateReputation updates the reputation based on uptime and honesty.
func UpdateReputation(rep *Reputation, uptime uint64, isHonest bool) *Reputation {
	if rep == nil {
		rep = NewReputation()
	}
	rep.Contributions += uptime
	if !isHonest {
		rep.CheatAttempts++
		rep.Score = math.Max(0.1, rep.Score*0.9)
	} else {
		rep.Score = math.Min(2.0, rep.Score+0.01*math.Log1p(float64(uptime)))
	}
	return rep
}

// DetectCheat checks for cheating based on computation volume.
func DetectCheat(computations uint64) bool {
	// Simple heuristic: flag as cheating if computations exceed a threshold
	const maxComputations = 100000
	return computations > maxComputations
}
