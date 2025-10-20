package core

import (
	"fmt"
	"sync"
)

// State manages the global state of the blockchain, users, and validators.
type State struct {
	Users      map[string]UserData
	Mutex      sync.Mutex
	Blockchain *TriadBlockchain
}

func NewState() *State {
	bc := NewTriadBlockchain()
	return &State{
		Users:      make(map[string]UserData),
		Blockchain: bc,
	}
}

func NewTriadBlockchain() *TriadBlockchain {
	consensus := NewConsensus()
	genesisBlock := NewBlock(0, []Transaction{}, "0", "genesis_validator")
	rootNode := &TriadNode{
		Block:    genesisBlock,
		Children: [3]*TriadNode{},
	}
	return &TriadBlockchain{
		Root:      rootNode,
		Consensus: consensus,
		Nodes:     map[string]*TriadNode{genesisBlock.Hash: rootNode},
	}
}

// AddBlock adds a new block to the triad tree.
func (s *State) AddBlock(transactions []Transaction, validator, parentHash string) {
	s.Blockchain.Mutex.Lock()
	defer s.Blockchain.Mutex.Unlock()

	// Verify validator
	if validator != s.Blockchain.Consensus.SelectValidator() {
		fmt.Printf("Invalid validator: %s\n", validator)
		return
	}

	// Find parent node
	parentNode, exists := s.Blockchain.Nodes[parentHash]
	if !exists {
		fmt.Printf("Parent block not found: %s\n", parentHash)
		return
	}

	// Check if parent can accept more children
	if len(parentNode.Children) >= 3 {
		fmt.Printf("Parent node has maximum children: %s\n", parentHash)
		return
	}

	newBlock := NewBlock(parentNode.Block.Index+1, transactions, parentHash, validator)
	newNode := &TriadNode{
		Block:    newBlock,
		Children: [3]*TriadNode{},
	}
	for i := 0; i < 3; i++ {
		if parentNode.Children[i] == nil {
			parentNode.Children[i] = newNode
			break
		}
	}
	s.Blockchain.Nodes[newBlock.Hash] = newNode
	fmt.Printf("Block added to triad tree: index=%d, hash=%s, validator=%s\n", newBlock.Index, newBlock.Hash, validator)
}

// ValidateBlockchain validates the triad blockchain.
func (s *State) ValidateBlockchain() bool {
	return s.Blockchain.ValidateTree()
}

// AddUser adds a new user to the state.
func (s *State) AddUser(address, deviceID string, data UserData) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	data.Balance = 1000
	data.LastNonce = 0
	data.Reputation = NewReputation()
	data.Devices = []string{deviceID}
	s.Users[address] = data
	s.Blockchain.Consensus.AddValidator(address, data.Balance, data.Reputation)
	return nil
}

// UpdateUser updates a user's data.
func (s *State) UpdateUser(address string, data UserData) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.Users[address] = data
	s.Blockchain.Consensus.AddValidator(address, data.Balance, data.Reputation)
}

// UpdateData updates a user's PoC contribution and trees planted.
func (s *State) UpdateData(address, deviceID string, contribution PoCContribution, trees int64) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	user, exists := s.Users[address]
	if !exists {
		return
	}
	user.PoCContribution = contribution
	user.TreesPlanted += trees
	s.Users[address] = user
}

// GetData retrieves a user's data.
func (s *State) GetData(address string) (UserData, bool) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	user, exists := s.Users[address]
	return user, exists
}

// GetTreesPlanted retrieves the number of trees planted by a user.
func (s *State) GetTreesPlanted(address string) int64 {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	user, exists := s.Users[address]
	if !exists {
		return 0
	}
	return user.TreesPlanted
}

// VerifyTransaction verifies a transaction.
func (s *State) VerifyTransaction(tx Transaction) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	user, exists := s.Users[tx.From]
	if !exists {
		return fmt.Errorf("user %s not found", tx.From)
	}
	if user.Balance < tx.Amount {
		return fmt.Errorf("insufficient balance")
	}
	if tx.Nonce <= user.LastNonce {
		return fmt.Errorf("invalid nonce")
	}
	// Add signature verification (simplified)
	return nil
}

// ExecuteTransaction executes a transaction.
func (s *State) ExecuteTransaction(from, to string, amount int64, nonce uint64) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	fromUser, exists := s.Users[from]
	if !exists {
		return fmt.Errorf("user %s not found", from)
	}
	if fromUser.Balance < amount {
		return fmt.Errorf("insufficient balance")
	}
	toUser, exists := s.Users[to]
	if !exists {
		return fmt.Errorf("recipient %s not found", to)
	}
	fromUser.Balance -= amount
	fromUser.LastNonce = nonce
	toUser.Balance += amount
	s.Users[from] = fromUser
	s.Users[to] = toUser
	s.Blockchain.Consensus.AddValidator(from, fromUser.Balance, fromUser.Reputation)
	s.Blockchain.Consensus.AddValidator(to, toUser.Balance, toUser.Reputation)
	return nil
}

// AddDevice adds a device to the user.
func (s *State) AddDevice(address, deviceID string) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	user, exists := s.Users[address]
	if !exists {
		return fmt.Errorf("user %s not found", address)
	}
	for _, id := range user.Devices {
		if id == deviceID {
			return fmt.Errorf("device already added")
		}
	}
	user.Devices = append(user.Devices, deviceID)
	s.Users[address] = user
	return nil
}

// RemoveDevice removes a device from the user.
func (s *State) RemoveDevice(address, deviceID string) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	user, exists := s.Users[address]
	if !exists {
		return fmt.Errorf("user %s not found", address)
	}
	for i, id := range user.Devices {
		if id == deviceID {
			user.Devices = append(user.Devices[:i], user.Devices[i+1:]...)
			s.Users[address] = user
			return nil
		}
	}
	return fmt.Errorf("device %s not found", deviceID)
}
