package main

// PoCContribution represents the Proof of Contribution metrics for a user
type PoCContribution struct {
	TransactionsValidated int `json:"transactionsValidated"`
	Computations          int `json:"computations"`
	AdsServed             int `json:"adsServed"`
	DataShared            int `json:"dataShared"`
	StorageProvided       int `json:"storageProvided"`
}

// UserData represents the data associated with a user in the Triad Network
type UserData struct {
	Address         string          `json:"address"`
	Balance         float64         `json:"balance"`
	PoCContribution PoCContribution `json:"pocContribution"`
}

// Transaction represents a transaction between two users
type Transaction struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Amount    float64 `json:"amount"`
	Timestamp int64   `json:"timestamp"`
}
