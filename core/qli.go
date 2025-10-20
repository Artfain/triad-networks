package core

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// QLI represents a Quality of Life Index for a user or node.
type QLI struct {
	Address     string
	Score       float64
	Timestamp   int64
	Computation uint64
	EcoActions  uint64
}

// CreateQLI creates a new QLI for a user.
func CreateQLI(address string, computation, ecoActions uint64) *QLI {
	qli := &QLI{
		Address:     address,
		Score:       calculateQLIScore(computation, ecoActions),
		Timestamp:   time.Now().UnixNano(),
		Computation: computation,
		EcoActions:  ecoActions,
	}
	return qli
}

// calculateQLIScore calculates the QLI score based on computation and eco actions.
func calculateQLIScore(computation, ecoActions uint64) float64 {
	// Simplified scoring function
	return float64(computation)/1000.0 + float64(ecoActions)*0.1
}

// Hash returns the hash of the QLI.
func (q *QLI) Hash() string {
	data := fmt.Sprintf("%s:%f:%d:%d:%d", q.Address, q.Score, q.Timestamp, q.Computation, q.EcoActions)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}
