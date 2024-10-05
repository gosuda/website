package main

import (
	"fmt"

	"gosuda.org/website/internal/types"
)

const (
	rootDir   = "root"
	publicDir = "public"
	distDir   = "dist"
	dbFile    = "zdata/data.json.zstd"
	baseURL   = "https://gosuda.org"
)

var (
	ErrInvalidMarkdown = fmt.Errorf("invalid markdown file")
)

type GenerationContext struct {
	DataStore *DataStore
	UsedPosts map[string]struct{}
	PathMap   map[string]string
}

type DataStore struct {
	Posts map[string]*types.Post `json:"posts"`
}
