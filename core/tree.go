package core

import (
	"fmt"
	"sync"
)

// TriadNode represents a node in the triad tree.
type TriadNode struct {
	Block    *Block
	Children [3]*TriadNode // Up to 3 children
}

// TriadTree represents the triad tree structure for the blockchain.
type TriadTree struct {
	Root  *TriadNode
	mutex sync.Mutex
}

// NewTriadTree creates a new triad tree with a genesis block.
func NewTriadTree() *TriadTree {
	genesisBlock := NewBlock(0, []Transaction{}, "0", "genesis_validator")
	return &TriadTree{
		Root: &TriadNode{
			Block:    genesisBlock,
			Children: [3]*TriadNode{},
		},
	}
}

// AddNode adds a new node to the triad tree.
func (t *TriadTree) AddNode(data []Transaction, validator string, parent *TriadNode) (*TriadNode, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if len(parent.Children) >= 3 {
		return nil, fmt.Errorf("parent node has maximum children")
	}

	newBlock := NewBlock(parent.Block.Index+1, data, parent.Block.Hash, validator)
	newNode := &TriadNode{
		Block:    newBlock,
		Children: [3]*TriadNode{},
	}

	for i := 0; i < 3; i++ {
		if parent.Children[i] == nil {
			parent.Children[i] = newNode
			break
		}
	}

	newBlock.Children = [3]string{}                                 // Initialize children hashes
	parent.Block.Children = t.updateChildrenHashes(parent.Children) // Update parent children hashes
	parent.Block.Hash = parent.Block.calculateHash()                // Recalculate parent hash

	return newNode, nil
}

// updateChildrenHashes updates the children hashes in the block.
func (t *TriadTree) updateChildrenHashes(children [3]*TriadNode) [3]string {
	var hashes [3]string
	for i, child := range children {
		if child != nil {
			hashes[i] = child.Block.Hash
		}
	}
	return hashes
}

// ValidateTree validates the triad tree.
func (t *TriadTree) ValidateTree() bool {
	return t.validateNode(t.Root, "0")
}

// validateNode recursively validates a node and its children.
func (t *TriadTree) validateNode(node *TriadNode, expectedParentHash string) bool {
	if node.Block.ParentHash != expectedParentHash {
		return false
	}
	if node.Block.Hash != node.Block.calculateHash() {
		return false
	}
	for _, child := range node.Children {
		if child != nil {
			if !t.validateNode(child, node.Block.Hash) {
				return false
			}
		}
	}
	return true
}
