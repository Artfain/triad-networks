package main

import (
	"errors"
	"sync"
)

// Определяем ошибки
var (
	ErrUserNotFound        = errors.New("user not found")
	ErrDeviceNotFound      = errors.New("device not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// State manages the global state of the Triad Network
type State struct {
	users   map[string]UserData
	devices map[string][]string // address -> device IDs
	nodes   []string
	tokens  map[string]float64
	mu      sync.Mutex
}

// NewState initializes a new State
func NewState() *State {
	return &State{
		users:   make(map[string]UserData),
		devices: make(map[string][]string),
		nodes:   []string{"node1", "node2", "node3"},
		tokens:  make(map[string]float64),
	}
}

// AddUser adds a new user to the state
func (s *State) AddUser(address string, data UserData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[address] = data
	s.tokens[address] = data.Balance
}

// UpdateUser updates an existing user's data
func (s *State) UpdateUser(address string, data UserData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[address] = data
	s.tokens[address] = data.Balance
}

// ExecuteTransaction performs a transaction between two users
func (s *State) ExecuteTransaction(from, to string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fromUser, exists := s.users[from]
	if !exists {
		return ErrUserNotFound
	}
	toUser, exists := s.users[to]
	if !exists {
		return ErrUserNotFound
	}
	if fromUser.Balance < amount {
		return ErrInsufficientBalance
	}

	fromUser.Balance -= amount
	toUser.Balance += amount
	s.users[from] = fromUser
	s.users[to] = toUser
	s.tokens[from] = fromUser.Balance
	s.tokens[to] = toUser.Balance
	return nil
}

// AddDevice associates a device with a user
func (s *State) AddDevice(address, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[address]; !exists {
		return ErrUserNotFound
	}
	s.devices[address] = append(s.devices[address], deviceID)
	return nil
}

// GetDevices returns the list of devices for a user
func (s *State) GetDevices(address string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[address]; !exists {
		return nil, ErrUserNotFound
	}
	return s.devices[address], nil
}

// RemoveDevice removes a device from a user
func (s *State) RemoveDevice(address, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[address]; !exists {
		return ErrUserNotFound
	}
	devices := s.devices[address]
	for i, id := range devices {
		if id == deviceID {
			s.devices[address] = append(devices[:i], devices[i+1:]...)
			return nil
		}
	}
	return ErrDeviceNotFound
}

// GetNodes returns the list of nodes
func (s *State) GetNodes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nodes
}

// GetTokens returns the token balance for a user
func (s *State) GetTokens(address string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokens[address]
}
