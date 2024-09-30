package main

import "gosuda.org/website/internal/types"

// Dummy data store for testing
type DataStore struct {
	Posts map[string]*types.Post `json:"posts"`
}
