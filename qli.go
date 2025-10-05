package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

func CreateQLI(address, deviceID string) (string, *ecdsa.PrivateKey, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate key: %v", err)
	}
	hash := sha256.Sum256([]byte(address + deviceID))
	qli := fmt.Sprintf("%x", hash)
	return qli, privKey, nil
}
