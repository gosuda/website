package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gosuda.org/website/internal/types"
	"gosuda.org/website/view/layout"
)

func TestAllRootMarkdownFilesValid(t *testing.T) {
	list, err := generateFileList(rootDir)
	if err != nil {
		t.Fatalf("failed to generate file list: %v", err)
	}

	count := 0
	for _, path := range list {
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			continue
		}
		count++

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		data = []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))

		doc, err := parseMarkdown(path, data)
		if err != nil {
			t.Errorf("failed to parse %s: %v", path, err)
			continue
		}
		if doc.Metadata.ID == "" {
			t.Errorf("%s has empty metadata ID", path)
		}
		if doc.Metadata.Title == "" {
			t.Errorf("%s has empty metadata Title", path)
		}
		if doc.Metadata.Path == "" {
			t.Errorf("%s has empty metadata Path", path)
		}
		if !doc.Metadata.NoTranslate && doc.Metadata.Description == "" {
			t.Errorf("%s has empty metadata Description", path)
		}
	}

	if count == 0 {
		t.Fatal("expected markdown files in rootDir, found 0")
	}
}

func TestCheckRetranslateTargets(t *testing.T) {
	tmpDir := t.TempDir()

	writeMD := func(name, id, title, body string, noTranslate bool) string {
		p := filepath.Join(tmpDir, name)
		noTransStr := ""
		if noTranslate {
			noTransStr = "no_translate: true\n"
		}
		content := "---\n" +
			"id: " + id + "\n" +
			"author: tester\n" +
			"title: " + title + "\n" +
			"description: test description\n" +
			"language: en\n" +
			"date: 2025-01-01T00:00:00Z\n" +
			"path: /blog/posts/" + id + "\n" +
			noTransStr +
			"---\n\n" + body + "\n"
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
		return p
	}

	// 1. Unchanged post with all languages translated
	p1 := writeMD("unchanged.md", "id-unchanged", "Unchanged Title", "Hello world.", false)
	data1, _ := os.ReadFile(p1)
	doc1, err := parseMarkdown(p1, data1)
	if err != nil {
		t.Fatalf("parse p1: %v", err)
	}
	if err := normalizeDocumentForHash(doc1); err != nil {
		t.Fatalf("normalize p1: %v", err)
	}
	allTranslated := make(map[string]*types.Document)
	for _, lang := range types.SupportedLanguages {
		allTranslated[string(lang)] = doc1
	}

	// 2. Modified post (hash changed)
	_ = writeMD("modified.md", "id-modified", "Modified Title", "Updated body.", false)

	// 3. New post (not in DataStore)
	_ = writeMD("new.md", "id-new", "New Title", "Brand new post.", false)

	// 4. NoTranslate post (should be ignored)
	_ = writeMD("notranslate.md", "id-notranslate", "No Translate", "Skip me.", true)

	ds := &DataStore{
		Posts: map[string]*types.Post{
			"id-unchanged": {
				ID:         "id-unchanged",
				FilePath:   p1,
				Path:       doc1.Metadata.Path,
				Hash:       doc1.Hash(),
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				Main:       doc1,
				Translated: allTranslated,
			},
			"id-modified": {
				ID:         "id-modified",
				FilePath:   filepath.Join(tmpDir, "modified.md"),
				Path:       "/blog/posts/id-modified",
				Hash:       "old-stale-hash",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				Main:       doc1,
				Translated: allTranslated,
			},
		},
	}

	targets, err := checkRetranslateTargets(ds, tmpDir)
	if err != nil {
		t.Fatalf("checkRetranslateTargets failed: %v", err)
	}

	if len(targets) != 2 {
		t.Fatalf("expected 2 retranslate targets, got %d: %+v", len(targets), targets)
	}

	reasons := map[string]string{}
	for _, target := range targets {
		reasons[target.ID] = target.Reason
	}
	if reasons["id-modified"] != "hash_changed" {
		t.Errorf("expected id-modified reason=hash_changed, got %q", reasons["id-modified"])
	}
	if reasons["id-new"] != "new_post" {
		t.Errorf("expected id-new reason=new_post, got %q", reasons["id-new"])
	}
}

func TestInjectOpenRouterCacheControl(t *testing.T) {
	reqBody := `{
		"model": "anthropic/claude-haiku-5.5",
		"messages": [
			{"role": "system", "content": "Static system instruction"},
			{"role": "user", "content": [
				{"type": "text", "text": "INPUT_TEXT:\nStatic source content"},
				{"type": "text", "text": "TARGET_LANGUAGE: Korean"}
			]}
		]
	}`

	updated, ok := injectOpenRouterCacheControl([]byte(reqBody))
	if !ok {
		t.Fatal("expected injectOpenRouterCacheControl to return true")
	}

	var parsed map[string]any
	if err := json.Unmarshal(updated, &parsed); err != nil {
		t.Fatalf("unmarshal updated: %v", err)
	}

	msgs := parsed["messages"].([]any)
	sysContent := msgs[0].(map[string]any)["content"].([]any)
	sysBlock := sysContent[0].(map[string]any)
	if sysBlock["cache_control"] == nil {
		t.Error("expected cache_control on system block")
	}

	userContent := msgs[1].(map[string]any)["content"].([]any)
	firstUserBlock := userContent[0].(map[string]any)
	secondUserBlock := userContent[1].(map[string]any)
	if firstUserBlock["cache_control"] == nil {
		t.Error("expected cache_control on first user content block (source text)")
	}
	if secondUserBlock["cache_control"] != nil {
		t.Error("did not expect cache_control on dynamic second user content block")
	}
}

func TestFormatLLMsTxtAndFooterLinks(t *testing.T) {
	enDoc := &types.Document{
		Metadata: types.Metadata{
			ID:          "post-1",
			Title:       "Test Blog Post",
			Description: "A detailed blog post.",
			Language:    "en",
			Date:        time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			Path:        "/blog/posts/test-blog-post",
		},
		Markdown: "# Test Blog Post\n\nHello GEO world.",
	}
	zhDoc := &types.Document{
		Metadata: types.Metadata{
			ID:          "post-1",
			Title:       "测试博客文章",
			Description: "一篇详细的博客文章。",
			Language:    "zh",
			Date:        time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			Path:        "/blog/posts/test-blog-post",
		},
		Markdown: "# 测试博客文章\n\n你好 GEO 世界。",
	}
	pkgDoc := &types.Document{
		Metadata: types.Metadata{
			ID:          "pkg-1",
			Title:       "Randflake ID",
			Description: "Distributed 64-bit ID generator.",
			Language:    "en",
			Date:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Path:        "/randflake",
			GoPackage:   "gosuda.org/randflake",
			GoRepoURL:   "https://github.com/gosuda/randflake.git",
		},
		Markdown: "# Randflake ID\n\nPackage docs.",
	}

	posts := []*types.Post{
		{
			ID:   "post-1",
			Path: "/blog/posts/test-blog-post",
			Main: enDoc,
			Translated: map[string]*types.Document{
				"en": enDoc,
				"zh": zhDoc,
			},
		},
		{
			ID:   "pkg-1",
			Path: "/randflake",
			Main: pkgDoc,
			Translated: map[string]*types.Document{
				"en": pkgDoc,
			},
		},
	}

	llmsEn := formatLLMsTxtForLang(posts, "en")
	if !strings.Contains(llmsEn, "[Randflake ID](https://gosuda.org/randflake) ([Markdown](https://gosuda.org/randflake.md))") {
		t.Errorf("expected package markdown link in English llms.txt, got:\n%s", llmsEn)
	}
	if !strings.Contains(llmsEn, "[Test Blog Post](https://gosuda.org/blog/posts/test-blog-post) ([Markdown](https://gosuda.org/blog/posts/test-blog-post.md))") {
		t.Errorf("expected blog post markdown link in English llms.txt, got:\n%s", llmsEn)
	}

	llmsZh := formatLLMsTxtForLang(posts, "zh")
	if !strings.Contains(llmsZh, "[测试博客文章](https://gosuda.org/zh/blog/posts/test-blog-post) ([Markdown](https://gosuda.org/zh/blog/posts/test-blog-post.md))") {
		t.Errorf("expected localized blog post link in Chinese llms.txt, got:\n%s", llmsZh)
	}

	llmsFull := formatLLMsFullTxt(posts)
	if !strings.Contains(llmsFull, "Go-Package: gosuda.org/randflake") || !strings.Contains(llmsFull, "Hello GEO world.") {
		t.Errorf("expected full corpus content in llms-full.txt, got:\n%s", llmsFull)
	}

	meta := &layout.Metadata{
		Language: "en",
		BaseURL:  "https://gosuda.org",
		Alternate: &layout.Alternate{
			Default: "https://gosuda.org/blog/posts/test-blog-post",
			Versions: []layout.KV{
				{Key: "en", Value: "https://gosuda.org/blog/posts/test-blog-post"},
				{Key: "zh", Value: "https://gosuda.org/zh/blog/posts/test-blog-post"},
				{Key: "ko", Value: "https://gosuda.org/ko/blog/posts/test-blog-post"},
			},
		},
	}
	var buf bytes.Buffer
	if err := layout.Footer(meta).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render footer: %v", err)
	}
	rendered := buf.String()
	for _, want := range []string{
		`href="/llms.txt"`,
		`href="/zh/blog/posts/test-blog-post"`,
		`hreflang="zh"`,
		`href="/ko/blog/posts/test-blog-post"`,
		`hreflang="ko"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("expected footer HTML to contain %q, got:\n%s", want, rendered)
		}
	}

	pkgMeta := &layout.Metadata{
		Language:    "en",
		Title:       "Randflake ID",
		Description: "Distributed 64-bit ID generator.",
		Author:      "Lemon",
		URL:         "https://gosuda.org/randflake",
		Canonical:   "https://gosuda.org/randflake",
		MarkdownURL: "https://gosuda.org/randflake.md",
		BaseURL:     "https://gosuda.org",
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		GoPackage:   "gosuda.org/randflake",
		GoRepoURL:   "https://github.com/gosuda/randflake.git",
	}
	var headBuf bytes.Buffer
	if err := layout.Head(pkgMeta).Render(context.Background(), &headBuf); err != nil {
		t.Fatalf("render head: %v", err)
	}
	headHTML := headBuf.String()
	for _, want := range []string{
		`<script type="application/ld+json">`,
		`"@type":"TechArticle"`,
		`"@type":"SoftwareSourceCode"`,
		`"codeRepository":"https://github.com/gosuda/randflake.git"`,
		`"programmingLanguage":"Go"`,
		`href="https://gosuda.org/.well-known/agent-card.json"`,
	} {
		if !strings.Contains(headHTML, want) {
			t.Errorf("expected head HTML to contain %q, got:\n%s", want, headHTML)
		}
	}

	agentCardBytes, err := formatAgentCardJSON(posts)
	if err != nil {
		t.Fatalf("formatAgentCardJSON: %v", err)
	}
	agentCardStr := string(agentCardBytes)
	for _, want := range []string{
		`"name": "GoSuda"`,
		`"documentationUrl": "https://gosuda.org/llms.txt"`,
		`"goPackage": "gosuda.org/randflake"`,
		`"markdownUrl": "https://gosuda.org/randflake.md"`,
	} {
		if !strings.Contains(agentCardStr, want) {
			t.Errorf("expected agent-card.json to contain %q, got:\n%s", want, agentCardStr)
		}
	}
}

