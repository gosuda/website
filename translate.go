package main

import (
	"context"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"gosuda.org/website/internal/markdown"
	"gosuda.org/website/internal/translate"
	"gosuda.org/website/internal/types"
)

func translatePost(_ *GenerationContext, post *types.Post, retranslate bool, ignoreLangs ...types.Lang) error {
	if post.Translated == nil {
		post.Translated = make(map[string]*types.Document)
	}

	if post.Main.Metadata.NoTranslate {
		return nil
	}

	if len(post.Main.Metadata.IgnoreLangs) > 0 {
		ignoreLangs = append([]string(nil), ignoreLangs...)
		ignoreLangs = append(ignoreLangs, post.Main.Metadata.IgnoreLangs...)
	}

	for _, lang := range post.Main.Metadata.IgnoreLangs {
		if lang == post.Main.Metadata.Language {
			continue
		}
		delete(post.Translated, lang)
	}

	ctx := context.Background()

	var langs []types.Lang
	if retranslate {
		// only retranslate the missing languages
		for _, lang := range types.SupportedLanguages {
			if _, ok := post.Translated[string(lang)]; !ok && !slices.Contains(ignoreLangs, lang) {
				langs = append(langs, lang)
			}
		}

		if len(langs) == 0 {
			return nil
		}
	} else {
		// all supported languages except the ones in ignoreLangs
		for _, lang := range types.SupportedLanguages {
			if !slices.Contains(ignoreLangs, lang) {
				langs = append(langs, lang)
			}
		}
	}

	for _, lang := range langs {
		log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("translating post")
		original := post.Main.Markdown
		original = strings.TrimPrefix(original, "---\n")
		_, origDocument, ok := strings.Cut(original, "---\n")
		if !ok {
			return ErrInvalidMarkdown
		}

		fullLangName := types.FullLangName(lang)

		meta := post.Main.Metadata
		meta.Language = lang

		log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("translating post title")
		newTitle, err := translate.Translate(ctx, llmModel, post.Main.Metadata.Title, fullLangName)
		if err != nil {
			return err
		}
		meta.Title = newTitle
		log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Str("title", newTitle).Msg("translated post title")

		log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("translating post description")
		newDescription, err := translate.Translate(ctx, llmModel, post.Main.Metadata.Description, fullLangName)
		if err != nil {
			return err
		}
		meta.Description = newDescription
		log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Str("description", newDescription).Msg("translated post description")

		log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("translating post content")
		tranDocument, err := translate.Translate(ctx, llmModel, origDocument, fullLangName)
		if err != nil {
			return err
		}
		log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("translated post content")

		newMeta, err := yaml.Marshal(&meta)
		if err != nil {
			return err
		}
		newDocument := "---\n" + string(newMeta) + "---\n" + tranDocument

		doc, err := markdown.ParseMarkdown(newDocument)
		if err != nil {
			return err
		}
		post.Translated[string(lang)] = doc
	}

	return nil
}
