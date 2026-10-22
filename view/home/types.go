package home

import "time"

type PostPreview struct {
	Title       string
	Author      string
	Description string
	Date        time.Time
	URL         string
	Path        string
	Cover       string
}

type Repo struct {
	Name        string
	Description string
	URL         string
	Language    string
	Stars       int
}

// GitHubStats is nil when the GitHub API could not be reached at build time.
type GitHubStats struct {
	RepoCount int
	StarCount int
	Repos     []Repo // sorted by stars, descending
}

type Stats struct {
	GitHub    *GitHubStats
	Articles  int
	Languages int
}
