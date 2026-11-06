package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/rs/zerolog/log"
	"github.com/zeebo/blake3"
	"gosuda.org/website/internal/types"
	"gosuda.org/website/view"
)

func langFeedID(id string, lang types.Lang) string {
	var buf [16]byte
	blake3.DeriveKey("LANGUAGE FEED ID v0.1 LANG:"+lang, []byte(id), buf[:])
	return hex.EncodeToString(buf[:])
}

func createFeedItem(post *types.Post, doc *types.Document, link string) *feeds.Item {
	return &feeds.Item{
		Id:          langFeedID(post.ID, doc.Metadata.Language),
		Title:       doc.Metadata.Title,
		Link:        &feeds.Link{Href: link},
		Author:      &feeds.Author{Name: doc.Metadata.Author},
		Description: doc.Metadata.Description,
		Created:     post.CreatedAt.UTC(),
		Updated:     post.UpdatedAt.UTC(),
	}
}

func generateGlobalFeed(gc *GenerationContext) error {
	log.Debug().Msg("start generating global RSS feed")
	globalFeed := &feeds.Feed{
		Title:       "Gosuda Blog",
		Link:        &feeds.Link{Href: "https://gosuda.org/"},
		Description: "Gosuda: A blog about software development, and other topics.",
		Author:      &feeds.Author{Name: "Gosuda", Email: "webmaster@gosuda.org"},
		Created:     time.Now().UTC(),
	}

	for _, post := range gc.DataStore.Posts {
		doc := post.Main
		if doc.Metadata.Language != "en" {
			enDoc, ok := post.Translated["en"]
			if !ok {
				continue
			}
			doc = enDoc
		}
		link := baseURL + post.Path

		globalFeed.Items = append(globalFeed.Items, createFeedItem(post, doc, link))
	}

	globalFeed.Items = append(globalFeed.Items, &feeds.Item{
		Id:          langFeedID("home", types.LangEnglish),
		Title:       "GoSuda | Home",
		Link:        &feeds.Link{Href: baseURL + "/"},
		Author:      &feeds.Author{Name: "GoSuda"},
		Description: "GoSuda is an industry-leading open source working group enabling developers to easily build, prototype, and deploy applications. Our comprehensive suite of tools and frameworks empowers developers to create robust, scalable solutions across various domains.",
		Created:     time.Date(2024, 10, 07, 0, 0, 0, 0, time.UTC),
		Updated:     time.Now().UTC(),
	})

	rss, err := globalFeed.ToRss()
	if err != nil {
		return err
	}

	sitemap, err := encodeSiteMapXML(globalFeed)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, "feed.rss"), []byte(rss), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, "en", "feed.rss"), []byte(rss), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, "sitemap.xml"), []byte(sitemap), 0644)
	if err != nil {
		return err
	}

	log.Debug().Msg("done generating global RSS feed")
	return nil
}

func generateLocalFeed(gc *GenerationContext, lang types.Lang) error {
	log.Debug().Str("lang", string(lang)).Msg("start generating local RSS feed")

	feed := &feeds.Feed{
		Title:       "GoSuda Blog" + " - " + types.FullLangName(lang),
		Link:        &feeds.Link{Href: baseURL + "/" + string(lang) + "/"},
		Description: "Gosuda: A blog about software development, and other topics.",
		Author:      &feeds.Author{Name: "Gosuda", Email: "webmaster@gosuda.org"},
		Created:     time.Now().UTC(),
	}

	for _, post := range gc.DataStore.Posts {
		doc, ok := post.Translated[string(lang)]
		if !ok {
			continue
		}
		link := baseURL + "/" + string(lang) + post.Path

		feed.Items = append(feed.Items, createFeedItem(post, doc, link))
	}

	feed.Items = append(feed.Items, &feeds.Item{
		Id:          langFeedID("home", lang),
		Title:       "GoSuda | Home",
		Link:        &feeds.Link{Href: baseURL + "/" + string(lang) + "/"},
		Author:      &feeds.Author{Name: "GoSuda"},
		Description: "GoSuda is an industry-leading open source working group enabling developers to easily build, prototype, and deploy applications. Our comprehensive suite of tools and frameworks empowers developers to create robust, scalable solutions across various domains.",
		Created:     time.Date(2024, 10, 07, 0, 0, 0, 0, time.UTC),
		Updated:     time.Now().UTC(),
	})

	rss, err := feed.ToRss()
	if err != nil {
		return err
	}

	sitemap, err := encodeSiteMapXML(feed)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, string(lang), "feed.rss"), []byte(rss), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, string(lang), "sitemap.xml"), sitemap, 0644)
	if err != nil {
		return err
	}

	log.Debug().Str("lang", string(lang)).Msg("done generating local RSS feed")
	return nil
}

const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"

func encodeSiteMapXML(feed *feeds.Feed) ([]byte, error) {
	var b bytes.Buffer
	err := view.Sitemap(feed).Render(context.Background(), &b)
	if err != nil {
		return nil, err
	}
	return append([]byte(xmlHeader), b.Bytes()...), nil
}

func generateLLMsTxt(gc *GenerationContext) error {
	log.Debug().Msg("start generating llms.txt indexes")

	posts := make([]*types.Post, 0, len(gc.DataStore.Posts))
	for _, post := range gc.DataStore.Posts {
		if post.Main == nil || post.Main.Metadata.Hidden {
			continue
		}
		posts = append(posts, post)
	}

	for _, lang := range types.SupportedLanguages {
		content := formatLLMsTxtForLang(posts, lang)
		langDir := filepath.Join(distDir, string(lang))
		if err := os.MkdirAll(langDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(langDir, "llms.txt"), []byte(content), 0644); err != nil {
			return err
		}
		if lang == types.LangEnglish {
			if err := os.WriteFile(filepath.Join(distDir, "llms.txt"), []byte(content), 0644); err != nil {
				return err
			}
		}
	}

	fullContent := formatLLMsFullTxt(posts)
	if err := os.WriteFile(filepath.Join(distDir, "llms-full.txt"), []byte(fullContent), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(distDir, "en", "llms-full.txt"), []byte(fullContent), 0644); err != nil {
		return err
	}

	agentCardJSON, err := formatAgentCardJSON(posts)
	if err != nil {
		return err
	}
	wellKnownDir := filepath.Join(distDir, ".well-known")
	if err := os.MkdirAll(wellKnownDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(wellKnownDir, "agent-card.json"), agentCardJSON, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(wellKnownDir, "agents.json"), agentCardJSON, 0644); err != nil {
		return err
	}

	log.Debug().Msg("done generating llms.txt indexes")
	return nil
}

func postDocForLang(post *types.Post, lang types.Lang) *types.Document {
	if post == nil {
		return nil
	}
	if post.Translated != nil {
		if doc, ok := post.Translated[string(lang)]; ok && doc != nil {
			return doc
		}
	}
	if post.Main != nil && post.Main.Metadata.Language == string(lang) {
		return post.Main
	}
	return nil
}

func isPackagePost(post *types.Post) bool {
	if post == nil {
		return false
	}
	if post.Main != nil && post.Main.Metadata.GoPackage != "" {
		return true
	}
	return !strings.HasPrefix(post.Path, "/blog/")
}

func formatLLMsTxtForLang(posts []*types.Post, lang types.Lang) string {
	var packages []*types.Post
	var blogPosts []*types.Post

	for _, post := range posts {
		if postDocForLang(post, lang) == nil {
			continue
		}
		if isPackagePost(post) {
			packages = append(packages, post)
		} else {
			blogPosts = append(blogPosts, post)
		}
	}

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Path < packages[j].Path
	})
	sort.Slice(blogPosts, func(i, j int) bool {
		if !blogPosts[i].Main.Metadata.Date.Equal(blogPosts[j].Main.Metadata.Date) {
			return blogPosts[i].Main.Metadata.Date.After(blogPosts[j].Main.Metadata.Date)
		}
		return blogPosts[i].Path < blogPosts[j].Path
	})

	langPrefix := ""
	homeLink := baseURL + "/"
	feedLink := baseURL + "/feed.rss"
	sitemapLink := baseURL + "/sitemap.xml"
	if lang != types.LangEnglish {
		langPrefix = "/" + string(lang)
		homeLink = baseURL + langPrefix + "/"
		feedLink = baseURL + langPrefix + "/feed.rss"
		sitemapLink = baseURL + langPrefix + "/sitemap.xml"
	}

	var sb strings.Builder
	sb.WriteString("# GoSuda\n\n")
	sb.WriteString("> GoSuda is an open-source working group building high-performance networking, storage, runtimes, and developer tooling in Go.\n\n")
	fmt.Fprintf(&sb, "- [Home](%s)\n", homeLink)
	sb.WriteString("- [GitHub Organization](https://github.com/gosuda)\n")
	fmt.Fprintf(&sb, "- [Full Markdown Corpus (llms-full.txt)](%s/llms-full.txt)\n", baseURL)
	fmt.Fprintf(&sb, "- [Agent Discovery Card (.well-known/agent-card.json)](%s/.well-known/agent-card.json)\n", baseURL)
	fmt.Fprintf(&sb, "- [RSS Feed](%s)\n", feedLink)
	fmt.Fprintf(&sb, "- [Sitemap](%s)\n\n", sitemapLink)

	if len(packages) > 0 {
		sb.WriteString("## Packages & Projects\n\n")
		for _, post := range packages {
			doc := postDocForLang(post, lang)
			pageURL := baseURL + langPrefix + post.Path
			mdURL := pageURL + ".md"
			if strings.HasSuffix(post.Path, "/") {
				mdURL = pageURL + "index.md"
			}
			desc := strings.TrimSpace(strings.ReplaceAll(doc.Metadata.Description, "\n", " "))
			if desc != "" {
				fmt.Fprintf(&sb, "- [%s](%s) ([Markdown](%s)): %s\n", doc.Metadata.Title, pageURL, mdURL, desc)
			} else {
				fmt.Fprintf(&sb, "- [%s](%s) ([Markdown](%s))\n", doc.Metadata.Title, pageURL, mdURL)
			}
		}
		sb.WriteString("\n")
	}

	if len(blogPosts) > 0 {
		sb.WriteString("## Blog Posts\n\n")
		for _, post := range blogPosts {
			doc := postDocForLang(post, lang)
			pageURL := baseURL + langPrefix + post.Path
			mdURL := pageURL + ".md"
			if strings.HasSuffix(post.Path, "/") {
				mdURL = pageURL + "index.md"
			}
			desc := strings.TrimSpace(strings.ReplaceAll(doc.Metadata.Description, "\n", " "))
			if desc != "" {
				fmt.Fprintf(&sb, "- [%s](%s) ([Markdown](%s)): %s\n", doc.Metadata.Title, pageURL, mdURL, desc)
			} else {
				fmt.Fprintf(&sb, "- [%s](%s) ([Markdown](%s))\n", doc.Metadata.Title, pageURL, mdURL)
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Localized Indexes\n\n")
	for _, l := range types.SupportedLanguages {
		lURL := baseURL + "/llms.txt"
		if l != types.LangEnglish {
			lURL = baseURL + "/" + string(l) + "/llms.txt"
		}
		fmt.Fprintf(&sb, "- [%s (%s)](%s)\n", types.FullLangName(l), l, lURL)
	}

	return sb.String()
}

func formatLLMsFullTxt(posts []*types.Post) string {
	var ordered []*types.Post
	for _, post := range posts {
		if postDocForLang(post, types.LangEnglish) != nil {
			ordered = append(ordered, post)
		}
	}

	sort.Slice(ordered, func(i, j int) bool {
		pkgI := isPackagePost(ordered[i])
		pkgJ := isPackagePost(ordered[j])
		if pkgI != pkgJ {
			return pkgI
		}
		if pkgI {
			return ordered[i].Path < ordered[j].Path
		}
		if !ordered[i].Main.Metadata.Date.Equal(ordered[j].Main.Metadata.Date) {
			return ordered[i].Main.Metadata.Date.After(ordered[j].Main.Metadata.Date)
		}
		return ordered[i].Path < ordered[j].Path
	})

	var sb strings.Builder
	sb.WriteString("# GoSuda Full Documentation & Blog Corpus\n\n")
	for i, post := range ordered {
		doc := postDocForLang(post, types.LangEnglish)
		if i > 0 {
			sb.WriteString("\n\n===\n\n")
		}
		fmt.Fprintf(&sb, "URL: %s%s\n", baseURL, post.Path)
		if post.Main.Metadata.GoPackage != "" {
			fmt.Fprintf(&sb, "Go-Package: %s (%s)\n", post.Main.Metadata.GoPackage, post.Main.Metadata.GoRepoURL)
		}
		sb.WriteString("\n")
		sb.WriteString(strings.TrimSpace(doc.Markdown))
		sb.WriteString("\n")
	}
	return sb.String()
}

type agentPackageResource struct {
	Name        string `json:"name"`
	GoPackage   string `json:"goPackage,omitempty"`
	Repository  string `json:"repository,omitempty"`
	URL         string `json:"url"`
	MarkdownURL string `json:"markdownUrl"`
	Description string `json:"description,omitempty"`
}

func formatAgentCardJSON(posts []*types.Post) ([]byte, error) {
	var packages []agentPackageResource
	for _, post := range posts {
		doc := postDocForLang(post, types.LangEnglish)
		if doc == nil || !isPackagePost(post) {
			continue
		}
		pageURL := baseURL + post.Path
		mdURL := pageURL + ".md"
		if strings.HasSuffix(post.Path, "/") {
			mdURL = pageURL + "index.md"
		}
		packages = append(packages, agentPackageResource{
			Name:        doc.Metadata.Title,
			GoPackage:   post.Main.Metadata.GoPackage,
			Repository:  post.Main.Metadata.GoRepoURL,
			URL:         pageURL,
			MarkdownURL: mdURL,
			Description: strings.TrimSpace(strings.ReplaceAll(doc.Metadata.Description, "\n", " ")),
		})
	}
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].URL < packages[j].URL
	})

	card := map[string]any{
		"schemaVersion":    "1.0",
		"name":             "GoSuda",
		"description":      "GoSuda is an open-source working group building high-performance networking, storage, runtimes, and developer tooling in Go.",
		"url":              baseURL + "/",
		"version":          "1.0.0",
		"documentationUrl": baseURL + "/llms.txt",
		"provider": map[string]string{
			"organization": "GoSuda",
			"url":          "https://github.com/gosuda",
		},
		"capabilities": map[string]bool{
			"streaming":              false,
			"pushNotifications":      false,
			"stateTransitionHistory": false,
		},
		"defaultInputModes":  []string{"text/plain", "text/markdown"},
		"defaultOutputModes": []string{"text/markdown", "text/html", "application/rss+xml"},
		"resources": map[string]string{
			"llmsTxt":             baseURL + "/llms.txt",
			"llmsFullTxt":         baseURL + "/llms-full.txt",
			"sitemapXml":          baseURL + "/sitemap.xml",
			"rssFeed":             baseURL + "/feed.rss",
			"markdownNegotiation": "Append .md to any article path or follow <link rel=\"alternate\" type=\"text/markdown\">",
		},
		"supportedLanguages": types.SupportedLanguages,
		"skills": []map[string]any{
			{
				"id":          "go-packages",
				"name":        "GoSuda Open-Source Go Packages",
				"description": "Documentation, vanity Go import paths, and source repositories for GoSuda libraries and tools.",
				"tags":        []string{"golang", "open-source", "networking", "storage", "cryptography", "ipc"},
				"packages":    packages,
			},
			{
				"id":          "engineering-blog",
				"name":        "GoSuda Engineering Blog",
				"description": "Technical articles and deep-dives on Go runtime internals, network protocols, and distributed systems in 23 languages.",
				"tags":        []string{"golang", "systems-programming", "blog", "documentation"},
				"indexUrl":    baseURL + "/llms.txt",
				"corpusUrl":   baseURL + "/llms-full.txt",
			},
		},
	}

	return json.MarshalIndent(card, "", "  ")
}
