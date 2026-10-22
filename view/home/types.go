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
	Category    string
}

type Category struct {
	Slug  string
	Label string
}

// Categories is the display order of the home filter. Slugs match the markdown `category` field.
var Categories = []Category{
	{"ai", "AI"},
	{"network", "Networking"},
	{"concurrency", "Concurrency"},
	{"database", "Databases"},
	{"lowlevel", "Low-level"},
	{"language", "Language"},
	{"security", "Security"},
	{"api", "APIs"},
	{"release", "Releases"},
	{"gui", "GUI"},
}

func CategoryLabel(slug string) string {
	for _, c := range Categories {
		if c.Slug == slug {
			return c.Label
		}
	}
	return ""
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
