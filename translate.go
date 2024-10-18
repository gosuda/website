package main

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"gosuda.org/website/internal/evaluate"
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
	if !retranslate {
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
		var retry int
		for retry < 3 {
			retry++
			if retry > 1 {
				log.Debug().Int("retry", retry).Str("path", post.FilePath).Str("lang", string(lang)).Msg("retrying translation")
				time.Sleep(time.Second * 3)
			}

			err := translateLang(ctx, post, lang)
			if err != nil {
				log.Error().Err(err).Str("path", post.FilePath).Str("lang", string(lang)).Msg("failed to translate, retrying")
				continue
			}
			break
		}
	}

	return nil
}

var ErrLowQualityTranslation = errors.New("low quality translation")

func translateLang(ctx context.Context, post *types.Post, lang types.Lang) error {
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
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("evaluating translated title")
	score, err := evaluate.EvaluateTranslation(ctx, llmModel, post.Main.Metadata.Language, lang, post.Main.Metadata.Title, newTitle)
	if err != nil {
		return err
	}
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Float64("score", score).Msg("evaluated translation")
	if score < 0.7 {
		return ErrLowQualityTranslation
	}

	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("translating post description")
	newDescription, err := translate.Translate(ctx, llmModel, post.Main.Metadata.Description, fullLangName)
	if err != nil {
		return err
	}
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("evaluating translated description")
	score, err = evaluate.EvaluateTranslation(ctx, llmModel, post.Main.Metadata.Language, lang, post.Main.Metadata.Description, newDescription)
	if err != nil {
		return err
	}
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Float64("score", score).Msg("evaluated translation")
	if score < 0.7 {
		return ErrLowQualityTranslation
	}

	meta.Description = newDescription
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Str("description", newDescription).Msg("translated post description")

	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("translating post content")
	tranDocument, err := translate.Translate(ctx, llmModel, origDocument, fullLangName)
	if err != nil {
		return err
	}
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("translated post content")

	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msg("evaluating translated post content")
	score, err = evaluate.EvaluateTranslation(ctx, llmModel, post.Main.Metadata.Language, lang, origDocument, tranDocument)
	if err != nil {
		return err
	}
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Float64("score", score).Msg("evaluated translation")
	if score < 0.7 {
		return ErrLowQualityTranslation
	}

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

	return nil
}
