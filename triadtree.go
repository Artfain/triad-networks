package main

import (
	"fmt"
	"sync"
)

type TriadTree struct {
	sync.Mutex
	users map[string]map[string]UserData
	qlis  map[string]string
}

func NewTriadTree() *TriadTree {
	return &TriadTree{
		users: make(map[string]map[string]UserData),
		qlis:  make(map[string]string),
	}
}

func (t *TriadTree) AddUser(address, deviceID string, userData UserData, qli string) error {
	t.Lock()
	defer t.Unlock()
	if _, exists := t.users[address]; !exists {
		t.users[address] = make(map[string]UserData)
	}
	t.users[address][deviceID] = userData
	t.qlis[address+":"+deviceID] = qli
	return nil
}

func (t *TriadTree) GetUser(address, deviceID string) (string, error) {
	t.Lock()
	defer t.Unlock()
	qli, exists := t.qlis[address+":"+deviceID]
	if !exists {
		return "", fmt.Errorf("user %s with device %s not found", address, deviceID)
	}
	return qli, nil
}
