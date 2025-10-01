package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pemistahl/lingua-go"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"gosuda.org/website/internal/description"
	"gosuda.org/website/internal/markdown"
	"gosuda.org/website/internal/translate"
	"gosuda.org/website/internal/types"
)

func parseMarkdown(path string, data []byte) (*types.Document, error) {
	log.Debug().Str("path", path).Msgf("rendering markdown file %s", path)
	doc, err := markdown.ParseMarkdown(string(data))
	if err != nil {
		return nil, err
	}
	log.Debug().Str("path", path).Int("rendered_size", len(doc.HTML)).Msgf("rendered markdown file %s", path)
	return doc, nil
}

func processMarkdownFile(gc *GenerationContext, path string) (*types.Document, error) {
	log.Debug().Str("path", path).Msgf("start processing markdown file %s", path)

	log.Debug().Str("path", path).Msgf("start reading markdown file %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")) // normalize line endings
	log.Debug().Str("path", path).Int("size", len(data)).Msgf("read markdown file %s", path)

	doc, err := parseMarkdown(path, data)
	if err != nil {
		return nil, err
	}

	if doc.Metadata.ID == "" {
		doc.Metadata.ID = types.RandID()
		log.Debug().Str("path", path).Str("id", doc.Metadata.ID).Msgf("assigned new ID to document %s", path)
	}

	if doc.Metadata.Date.IsZero() {
		doc.Metadata.Date = time.Now().UTC()
		log.Debug().Str("path", path).Msgf("assigned new date to document %s", path)
	}

	if doc.Metadata.Path == "" {
		doc.Metadata.Path = generatePath(doc.Metadata.Title)
	}

	if doc.Metadata.Description == "" && llmModel != nil {
		log.Debug().Str("path", path).Msgf("generating description for document %s", path)
		desc, err := description.GenerateDescription(context.Background(), llmModel, doc.Markdown)
		if err != nil {
			log.Error().Str("path", path).Err(err).Msgf("failed to generate description for document %s", path)
		}
		doc.Metadata.Description = desc
		log.Debug().Str("path", path).Str("description", doc.Metadata.Description).Msgf("generated description for document %s", path)
	}

	if doc.Metadata.Language == "" {
		log.Debug().Str("path", path).Msgf("detecting language of document %s", path)
		detectedLang, ok := languageDetector.DetectLanguageOf(doc.Markdown)
		lang := "en"
		if ok {
			lang = mapDetectedLanguage(detectedLang)
			confidence := languageDetector.ComputeLanguageConfidence(doc.Markdown, detectedLang)
			log.Debug().Str("path", path).Str("lang", lang).Float64("confidence", confidence).Msgf("detected language of document %s", path)
			doc.Metadata.Language = lang
		}
	}

	log.Debug().Str("path", path).Msgf("saving updated document %s", path)

	if doc.Type == types.DocumentTypeMarkdown {
		newMeta, err := yaml.Marshal(&doc.Metadata)
		if err != nil {
			return nil, err
		}

		original := doc.Markdown
		original = strings.TrimPrefix(original, "---\n")
		_, origDocument, ok := strings.Cut(original, "---\n")
		if !ok {
			return nil, ErrInvalidMarkdown
		}
		newDocument := "---\n" + string(newMeta) + "---\n" + origDocument
		doc.Markdown = newDocument

		fStat, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		err = os.WriteFile(path, []byte(doc.Markdown), fStat.Mode())
		if err != nil {
			return nil, err
		}
		log.Debug().Str("path", path).Msgf("saved updated document %s", path)
	} else {
		log.Debug().Str("path", path).Msgf("skipping non-markdown document %s", path)
	}

	now := time.Now()

	// Update Post Object
	var post *types.Post
	if p, ok := gc.DataStore.Posts[doc.Metadata.ID]; ok {
		post = p
	} else {
		post = &types.Post{
			ID:         doc.Metadata.ID,
			CreatedAt:  now,
			UpdatedAt:  now,
			Translated: make(map[string]*types.Document),
		}
		gc.DataStore.Posts[doc.Metadata.ID] = post
	}

	hash := doc.Hash()
	post.FilePath = path
	post.Path = doc.Metadata.Path
	post.Main = doc
	if post.Translated == nil {
		post.Translated = make(map[string]*types.Document)
	}
	post.Translated[doc.Metadata.Language] = doc

	if llmModel != nil {
		if post.Hash != hash {
			post.Hash = hash
			post.UpdatedAt = now
			err = translatePost(gc, post, true, doc.Metadata.Language)
			if err != nil {
				log.Error().Str("path", path).Err(err).Msg("failed to translate")
			}
		} else {
			err = translatePost(gc, post, false, doc.Metadata.Language)
			if err != nil {
				log.Error().Str("path", path).Err(err).Msg("failed to translate")
			}
		}
	}

	if gc.UsedPosts == nil {
		gc.UsedPosts = make(map[string]struct{})
	}
	gc.UsedPosts[post.ID] = struct{}{}

	if gc.PathMap == nil {
		gc.PathMap = make(map[string]string)
	}
	gc.PathMap[post.Path] = post.ID

	log.Debug().Str("path", path).Msgf("done processing markdown file %s", path)
	return doc, nil
}

func generatePath(title string) string {
	lang, ok := languageDetector.DetectLanguageOf(title)
	if !ok {
		lang = lingua.English
	}
	langCode := mapDetectedLanguage(lang)
	log.Debug().Str("title", title).Str("lang", langCode).Msgf("detected language of title %s", title)

	if llmModel != nil && langCode != "en" {
		var retries int
		for retries < 3 {
			retries++
			translatedTitle, err := translate.Translate(context.Background(), llmModel, title, types.FullLangName("en"))
			if err != nil {
				log.Error().Err(err).Str("title", title).Msg("failed to translate title")
				time.Sleep(time.Second * 2)
				continue
			}
			log.Debug().Str("title", title).Str("lang", langCode).Str("translatedTitle", translatedTitle).Msgf("translated title %q", title)
			title = translatedTitle
			break
		}
	}

	title = strings.TrimSpace(title)
	fp := strings.TrimPrefix(title, rootDir)
	for strings.HasPrefix(fp, "/") {
		fp = strings.TrimPrefix(fp, "/")
	}

	fp = strings.ToLower(fp)
	fp = strings.ReplaceAll(fp, " ", "-")
	fp = strings.ReplaceAll(fp, "/", "-")
	fp = strings.ReplaceAll(fp, `{`, "-")
	fp = strings.ReplaceAll(fp, `}`, "-")
	fp = strings.ReplaceAll(fp, `|`, "-")
	fp = strings.ReplaceAll(fp, `\`, "-")
	fp = strings.ReplaceAll(fp, `^`, "-")
	fp = strings.ReplaceAll(fp, `~`, "-")
	fp = strings.ReplaceAll(fp, `[`, "-")
	fp = strings.ReplaceAll(fp, `]`, "-")
	fp = strings.ReplaceAll(fp, `'`, "-")
	fp = strings.ReplaceAll(fp, `"`, "-")
	fp = strings.ReplaceAll(fp, "`", "-")
	fp = strings.ReplaceAll(fp, ",", "-")
	fp = strings.ReplaceAll(fp, ".", "-")
	fp = strings.ReplaceAll(fp, "?", "-")
	fp = strings.ReplaceAll(fp, "&", "-")
	fp = strings.ReplaceAll(fp, "=", "-")

	fp = strings.ReplaceAll(fp, `----`, "-")
	fp = strings.ReplaceAll(fp, `---`, "-")
	fp = strings.ReplaceAll(fp, `--`, "-")
	fp = strings.ReplaceAll(fp, `--`, "-")
	fp = strings.ReplaceAll(fp, `--`, "-")
	fp = strings.TrimSuffix(fp, "---")
	fp = strings.TrimSuffix(fp, "--")
	fp = strings.TrimSuffix(fp, "-")

	var b [4]byte
	rand.Read(b[:])
	fp = fmt.Sprintf("/blog/posts/%s-z%x", fp, b)

	return fp
}
