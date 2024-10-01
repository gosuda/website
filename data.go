package main

import "gosuda.org/website/internal/types"

type GenerationContext struct {
	DataStore *DataStore
}

type DataStore struct {
	Posts map[string]*types.Post `json:"posts"`
}
