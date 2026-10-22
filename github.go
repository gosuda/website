package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"gosuda.org/website/view/home"
)

// fetchGitHubStats lists the org's public, non-fork repositories.
// GITHUB_TOKEN is optional; it only raises the API rate limit.
func fetchGitHubStats(org string) (*home.GitHubStats, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	stats := &home.GitHubStats{}

	for page := 1; ; page++ {
		req, err := http.NewRequest("GET", "https://api.github.com/orgs/"+org+"/repos?type=public&per_page=100&page="+strconv.Itoa(page), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		var repos []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			HTMLURL     string `json:"html_url"`
			Language    string `json:"language"`
			Stars       int    `json:"stargazers_count"`
			Fork        bool   `json:"fork"`
			Archived    bool   `json:"archived"`
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("github api: %s", resp.Status)
		}
		err = json.NewDecoder(resp.Body).Decode(&repos)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		for _, r := range repos {
			if r.Fork {
				continue
			}
			stats.RepoCount++
			stats.StarCount += r.Stars
			if !r.Archived {
				stats.Repos = append(stats.Repos, home.Repo{
					Name:        r.Name,
					Description: r.Description,
					URL:         r.HTMLURL,
					Language:    r.Language,
					Stars:       r.Stars,
				})
			}
		}
		if len(repos) < 100 {
			break
		}
	}

	sort.SliceStable(stats.Repos, func(i, j int) bool {
		return stats.Repos[i].Stars > stats.Repos[j].Stars
	})
	return stats, nil
}
