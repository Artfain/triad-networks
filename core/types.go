package core

import "sync"

// UserData represents user data in the blockchain.
type UserData struct {
	Balance         int64
	LastNonce       uint64
	Reputation      *Reputation
	Devices         []string
	PoCContribution PoCContribution
	TreesPlanted    int64
}

// Transaction represents a blockchain transaction.
type Transaction struct {
	From      string
	To        string
	Amount    int64
	Timestamp int64
	Nonce     uint64
	PrevHash  string
	Signature string
}

// PoCContribution represents proof-of-contribution metrics.
type PoCContribution struct {
	Computations uint64
	Storage      float64
	Bandwidth    float64
	Uptime       uint64
	EcoActions   uint64
}

// TriadBlockchain represents the blockchain as a triad tree.
type TriadBlockchain struct {
	Root      *TriadNode
	Mutex     sync.Mutex
	Consensus *Consensus
	Nodes     map[string]*TriadNode // Map of hash to node for quick lookup
}

// ValidateTree validates the triad blockchain.
func (bc *TriadBlockchain) ValidateTree() bool {
	return bc.Root != nil && bc.validateNode(bc.Root, "0")
}

// validateNode recursively validates a node and its children.
func (bc *TriadBlockchain) validateNode(node *TriadNode, expectedParentHash string) bool {
	if node.Block.ParentHash != expectedParentHash {
		return false
	}
	if node.Block.Hash != node.Block.calculateHash() {
		return false
	}
	for _, child := range node.Children {
		if child != nil {
			if !bc.validateNode(child, node.Block.Hash) {
				return false
			}
		}
	}
	return true
}
