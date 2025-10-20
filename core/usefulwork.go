package core

import (
	"math/rand"
	"time"
)

// PerformUsefulWork simulates a useful computation for PoC.
func PerformUsefulWork(computations uint64) uint64 {
	// Simulate some computation (e.g., Monte Carlo simulation for pi)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	inside := uint64(0)
	for i := uint64(0); i < computations; i++ {
		x := r.Float64()
		y := r.Float64()
		if x*x+y*y <= 1 {
			inside++
		}
	}
	return inside
}
