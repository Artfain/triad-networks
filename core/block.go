package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Block represents a block in the triad-based blockchain.
type Block struct {
	Index      int
	Timestamp  int64
	Data       []Transaction // Transactions or PoC data
	ParentHash string        // Hash of the parent block
	Hash       string        // Hash of this block
	Children   [3]string     // Hashes of up to 3 child blocks
	Validator  string        // Address of the validator
	Signature  string        // Signature of the block
}

func NewBlock(index int, data []Transaction, parentHash string, validator string) *Block {
	b := &Block{
		Index:      index,
		Timestamp:  time.Now().UnixNano(),
		Data:       data,
		ParentHash: parentHash,
		Children:   [3]string{},
		Validator:  validator,
	}
	b.Hash = b.calculateHash()
	return b
}

// calculateHash calculates the hash of the block.
func (b *Block) calculateHash() string {
	data, _ := json.Marshal(struct {
		Index      int
		Timestamp  int64
		Data       []Transaction
		ParentHash string
		Children   [3]string
		Validator  string
	}{
		Index:      b.Index,
		Timestamp:  b.Timestamp,
		Data:       b.Data,
		ParentHash: b.ParentHash,
		Children:   b.Children,
		Validator:  b.Validator,
	})
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// SignBlock signs the block with the validator's private key (simplified, use actual ECDSA in production).
func (b *Block) SignBlock(privateKey string) {
	b.Signature = "signed_" + b.Hash // Simplified signature, replace with real ECDSA
}
