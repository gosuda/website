package layout

import (
	"encoding/json"
	"strings"
	"time"
)

type Metadata struct {
	Language    string
	Title       string
	Description string
	Author      string
	Keywords    []string
	Image       string
	URL         string
	BaseURL     string
	Canonical   string
	MarkdownURL string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	GoImport    string
	GoPackage   string
	GoRepoURL   string
	Category    string
	IsIndex     bool
	CustomHead  string

	Alternate *Alternate
}

type Alternate struct {
	Default  string
	Versions []KV
}

type KV struct {
	Key   string
	Value string
}

// HomeURL returns the localized home page path.
func (m *Metadata) HomeURL() string {
	if m == nil || m.Language == "" || m.Language == "en" {
		return "/"
	}
	return "/" + m.Language + "/"
}

// FeedURL returns the localized RSS feed path.
func (m *Metadata) FeedURL() string {
	if m == nil || m.Language == "" || m.Language == "en" {
		return "/feed.rss"
	}
	return "/" + m.Language + "/feed.rss"
}

// LLMsTxtURL returns the localized llms.txt index path.
func (m *Metadata) LLMsTxtURL() string {
	if m == nil || m.Language == "" || m.Language == "en" {
		return "/llms.txt"
	}
	return "/" + m.Language + "/llms.txt"
}

// AlternatePath converts a full alternate URL into a root-relative path so
// static language links work across production, preview, and local builds.
func (m *Metadata) AlternatePath(fullURL string) string {
	if m != nil && m.BaseURL != "" && strings.HasPrefix(fullURL, m.BaseURL) {
		rel := strings.TrimPrefix(fullURL, m.BaseURL)
		if rel == "" {
			return "/"
		}
		return rel
	}
	return fullURL
}

// JSONLD returns a Schema.org JSON-LD representation of the page metadata for
// search engines and generative AI agents.
func (m *Metadata) JSONLD() string {
	if m == nil || m.Title == "" {
		return ""
	}

	baseURL := m.BaseURL
	if baseURL == "" {
		baseURL = "https://gosuda.org"
	}

	publisher := map[string]any{
		"@type":  "Organization",
		"name":   "GoSuda",
		"url":    baseURL,
		"sameAs": []string{"https://github.com/gosuda"},
	}

	isHome := m.IsIndex || m.URL == baseURL+"/" || (m.Language != "" && m.URL == baseURL+"/"+m.Language+"/")
	if isHome {
		site := map[string]any{
			"@context":    "https://schema.org",
			"@type":       "WebSite",
			"name":        "GoSuda",
			"headline":    m.Title,
			"description": m.Description,
			"url":         m.URL,
			"inLanguage":  m.Language,
			"publisher":   publisher,
		}
		data, err := json.Marshal(site)
		if err != nil {
			return ""
		}
		return string(data)
	}

	canonical := m.Canonical
	if canonical == "" {
		canonical = m.URL
	}

	author := map[string]any{
		"@type": "Organization",
		"name":  "GoSuda",
		"url":   baseURL,
	}
	if m.Author != "" && !strings.EqualFold(m.Author, "GoSuda") {
		author = map[string]any{
			"@type": "Person",
			"name":  m.Author,
		}
	}

	article := map[string]any{
		"@type":            "TechArticle",
		"headline":         m.Title,
		"description":      m.Description,
		"url":              m.URL,
		"mainEntityOfPage": canonical,
		"inLanguage":       m.Language,
		"author":           author,
		"publisher":        publisher,
	}
	if m.Image != "" {
		article["image"] = m.Image
	}
	if !m.CreatedAt.IsZero() {
		article["datePublished"] = m.CreatedAt.UTC().Format(time.RFC3339)
	}
	if !m.UpdatedAt.IsZero() {
		article["dateModified"] = m.UpdatedAt.UTC().Format(time.RFC3339)
	}
	if m.Category != "" {
		article["articleSection"] = m.Category
	}

	if m.GoPackage != "" || m.GoRepoURL != "" {
		pkgName := m.GoPackage
		if pkgName == "" {
			pkgName = m.Title
		}
		software := map[string]any{
			"@type":               "SoftwareSourceCode",
			"name":                pkgName,
			"description":         m.Description,
			"url":                 canonical,
			"programmingLanguage": "Go",
			"runtimePlatform":     "Go",
			"maintainer":          publisher,
		}
		if m.GoRepoURL != "" {
			software["codeRepository"] = m.GoRepoURL
		}
		graph := map[string]any{
			"@context": "https://schema.org",
			"@graph":   []any{article, software},
		}
		data, err := json.Marshal(graph)
		if err != nil {
			return ""
		}
		return string(data)
	}

	article["@context"] = "https://schema.org"
	data, err := json.Marshal(article)
	if err != nil {
		return ""
	}
	return string(data)
}

