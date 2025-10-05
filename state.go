package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
)

type State struct {
	users map[string]UserData
	nodes []string
	mutex sync.Mutex
}

func NewState() *State {
	return &State{
		users: make(map[string]UserData),
	}
}

func (s *State) AddUser(address, deviceID string, data UserData, tree *TriadTree, qli string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data.Balance = 1000
	data.LastNonce = 0
	data.Reputation = NewReputation()
	data.Devices = []string{deviceID}
	s.users[address] = data

	if err := tree.AddUser(address, deviceID, data, qli); err != nil {
		return fmt.Errorf("failed to add user to TriadTree: %v", err)
	}
	if err := StoreData(address, deviceID, data); err != nil {
		return fmt.Errorf("failed to store user data: %v", err)
	}
	return nil
}

func (s *State) UpdateUser(address string, data UserData) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.users[address] = data
}

func (s *State) VerifyTransaction(tx Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	user, exists := s.users[tx.From]
	if !exists {
		return fmt.Errorf("user %s not found", tx.From)
	}
	if user.Balance < tx.Amount {
		return fmt.Errorf("insufficient balance")
	}
	if tx.Nonce <= user.LastNonce {
		return fmt.Errorf("invalid nonce")
	}
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s%s%d%d%d%s", tx.From, tx.To, tx.Amount, tx.Timestamp, tx.Nonce, tx.PrevHash)))
	sigBytes, err := hex.DecodeString(tx.Signature)
	if err != nil {
		return fmt.Errorf("decode signature: %v", err)
	}
	sigR := new(big.Int).SetBytes(sigBytes[:len(sigBytes)/2])
	sigS := new(big.Int).SetBytes(sigBytes[len(sigBytes)/2:])
	pubKeyBytes, err := hex.DecodeString(user.PublicKey)
	if err != nil {
		return fmt.Errorf("decode public key: %v", err)
	}
	x := new(big.Int).SetBytes(pubKeyBytes[:len(pubKeyBytes)/2])
	y := new(big.Int).SetBytes(pubKeyBytes[len(pubKeyBytes)/2:])
	pubKey := &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
	if !ecdsa.Verify(pubKey, hash[:], sigR, sigS) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func (s *State) ExecuteTransaction(from, to string, amount int64, nonce uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	fromUser, exists := s.users[from]
	if !exists {
		return fmt.Errorf("user %s not found", from)
	}
	if fromUser.Balance < amount {
		return fmt.Errorf("insufficient balance")
	}
	toUser, exists := s.users[to]
	if !exists {
		return fmt.Errorf("recipient %s not found", to)
	}
	fromUser.Balance -= amount
	fromUser.LastNonce = nonce
	toUser.Balance += amount
	s.users[from] = fromUser
	s.users[to] = toUser
	return nil
}

func (s *State) AddDevice(address, deviceID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	user, exists := s.users[address]
	if !exists {
		return fmt.Errorf("user %s not found", address)
	}
	for _, id := range user.Devices {
		if id == deviceID {
			return fmt.Errorf("device already added")
		}
	}
	user.Devices = append(user.Devices, deviceID)
	s.users[address] = user
	return nil
}

func (s *State) RemoveDevice(address, deviceID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	user, exists := s.users[address]
	if !exists {
		return fmt.Errorf("user %s not found", address)
	}
	for i, id := range user.Devices {
		if id == deviceID {
			user.Devices = append(user.Devices[:i], user.Devices[i+1:]...)
			s.users[address] = user
			return nil
		}
	}
	return fmt.Errorf("device not found")
}

func (s *State) GetDevices(address string) ([]string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	user, exists := s.users[address]
	if !exists {
		return nil, fmt.Errorf("user %s not found", address)
	}
	return user.Devices, nil
}

func (s *State) AddNode(node string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, n := range s.nodes {
		if n == node {
			return
		}
	}
	s.nodes = append(s.nodes, node)
}

func (s *State) GetNodes() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.nodes
}

func (s *State) GetTokens(address string) int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	user, exists := s.users[address]
	if !exists {
		return 0
	}
	return user.Balance
}

func (s *State) GetParticipantCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.users)
}

func (s *State) UpdateReputation(address string, uptime uint64, isHonest bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	user, exists := s.users[address]
	if !exists {
		return
	}
	user.Reputation = UpdateReputation(user.Reputation, uptime, isHonest)
	s.users[address] = user
}
