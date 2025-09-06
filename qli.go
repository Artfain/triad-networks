package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// CreateQLI generates a Quantum Ledger Identifier for a user
func CreateQLI(data UserData) (string, error) {
	input := fmt.Sprintf("%s:%f:%d:%d:%d:%d:%d",
		data.Address,
		data.Balance,
		data.PoCContribution.TransactionsValidated,
		data.PoCContribution.Computations,
		data.PoCContribution.AdsServed,
		data.PoCContribution.DataShared,
		data.PoCContribution.StorageProvided,
	)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:]), nil
}
