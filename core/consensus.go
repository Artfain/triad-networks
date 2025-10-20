package core

import (
	"math/rand"
	"sync"
)

// Consensus manages the PoS + PoC consensus mechanism.
type Consensus struct {
	validators map[string]int64       // Address -> Stake (balance)
	reputation map[string]*Reputation // Address -> Reputation
	mutex      sync.Mutex
}

// NewConsensus creates a new consensus instance.
func NewConsensus() *Consensus {
	return &Consensus{
		validators: make(map[string]int64),
		reputation: make(map[string]*Reputation),
	}
}

// AddValidator adds a validator with stake and reputation.
func (c *Consensus) AddValidator(address string, stake int64, rep *Reputation) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.validators[address] = stake
	c.reputation[address] = rep
}

// RemoveValidator removes a validator.
func (c *Consensus) RemoveValidator(address string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.validators, address)
	delete(c.reputation, address)
}

// SelectValidator selects a validator based on stake and reputation-weighted random selection.
func (c *Consensus) SelectValidator() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.validators) == 0 {
		return ""
	}

	totalWeight := float64(0)
	for address, stake := range c.validators {
		repScore := c.reputation[address].Score
		totalWeight += float64(stake) * repScore
	}

	if totalWeight == 0 {
		return ""
	}

	randNum := rand.Float64()
	randNum *= totalWeight
	currentWeight := float64(0)
	for address, stake := range c.validators {
		repScore := c.reputation[address].Score
		currentWeight += float64(stake) * repScore
		if randNum < currentWeight {
			return address
		}
	}
	return ""
}

// ValidatePoC validates the Proof-of-Contribution.
func (c *Consensus) ValidatePoC(address string, contribution PoCContribution) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	rep, exists := c.reputation[address]
	if !exists {
		return false
	}
	// Simple PoC validation based on contribution and reputation
	if contribution.Computations > 0 && rep.Score > 0.5 {
		return true
	}
	return false
}
