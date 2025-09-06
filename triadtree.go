package main

import "errors"

// TriadTree represents the structure for managing users in a triad-based tree
type TriadTree struct {
	users map[string]UserData
	qlis  map[string]string
}

// NewTriadTree initializes a new TriadTree
func NewTriadTree() *TriadTree {
	return &TriadTree{
		users: make(map[string]UserData),
		qlis:  make(map[string]string),
	}
}

// AddUser adds a user to the TriadTree with their QLI
func (t *TriadTree) AddUser(address string, data UserData, qli string) error {
	t.users[address] = data
	t.qlis[address] = qli
	return nil
}

// GetUser retrieves the QLI for a user
func (t *TriadTree) GetUser(address string) (string, error) {
	qli, exists := t.qlis[address]
	if !exists {
		return "", errors.New("user not found in triad tree")
	}
	return qli, nil
}
