package main

import "gosuda.org/website/internal/types"

type GenerationContext struct {
	DataStore *DataStore `json:"datastore"`

	PathMap   map[string]string `json:"path_map"`   // key is the path, value is the post ID
	UsedPosts map[string]bool   `json:"used_posts"` // key is the post ID, value is true if the post is used
}

type DataStore struct {
	Posts map[string]*types.Post `json:"posts"`
}
