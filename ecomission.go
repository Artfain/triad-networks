package main

import "sync"

var (
	treesPlanted int
	mu           sync.Mutex
)

// UpdateTreesPlanted updates the global count of trees planted
func UpdateTreesPlanted(computations int) error {
	mu.Lock()
	defer mu.Unlock()
	treesPlanted += computations / 100 // Example: 1 tree per 100 computations
	return nil
}

// GetTreesPlanted returns the total number of trees planted
func GetTreesPlanted() (int, error) {
	mu.Lock()
	defer mu.Unlock()
	return treesPlanted, nil
}
