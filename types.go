package main

type PoCContribution struct {
	Computations uint64 `json:"computations"`
	Storage      uint64 `json:"storage"`
	Bandwidth    uint64 `json:"bandwidth"`
	Uptime       uint64 `json:"uptime"`
	EcoActions   uint64 `json:"ecoActions"`
	Challenge    string `json:"challenge"`
	QLI          string `json:"qli"`
	Action       string `json:"action"`
}

type UserData struct {
	Address         string          `json:"address"`
	Balance         int64           `json:"balance"`
	PoCContribution PoCContribution `json:"pocContribution"`
	Devices         []string        `json:"devices"`
	PublicKey       string          `json:"publicKey"`
	LastNonce       uint64          `json:"lastNonce"`
	Reputation      *Reputation     `json:"reputation"`
}

type Transaction struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    int64  `json:"amount"`
	Timestamp int64  `json:"timestamp"`
	Nonce     uint64 `json:"nonce"`
	Signature string `json:"signature"`
	PrevHash  string `json:"prevHash"`
}

type Message struct {
	Action      string      `json:"action"`
	UserData    UserData    `json:"userData"`
	DeviceID    string      `json:"deviceID"`
	Transaction Transaction `json:"transaction"`
	LoadLimit   uint64      `json:"loadLimit"`
	Power       struct {
		CPUPercent float64 `json:"cpuPercent"`
		MemoryMB   float64 `json:"memoryMB"`
	} `json:"power"`
	MFAToken   string  `json:"mfaToken"`
	CPULoad    float64 `json:"cpuLoad"`
	Storage    float64 `json:"storage"`
	Bandwidth  float64 `json:"bandwidth"`
	Uptime     uint64  `json:"uptime"`
	EcoActions uint64  `json:"ecoActions"`
}
