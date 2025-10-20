package core

import (
	"fmt"
	"time"
)

// AddUser adds a user to the triad tree (simplified, extend as needed).
func (t *TriadTree) AddUser(address, deviceID string, data UserData, qli string) error {
	// Simplified: store user data in a block
	transactions := []Transaction{
		{
			From:      "system",
			To:        address,
			Amount:    data.Balance,
			Timestamp: time.Now().UnixNano(),
			Nonce:     data.LastNonce,
			PrevHash:  t.Root.Block.Hash,
			Signature: "system_signature",
		},
	}
	_, err := t.AddNode(transactions, "genesis_validator", t.Root)
	if err != nil {
		return fmt.Errorf("failed to add user block: %v", err)
	}
	return nil
}
