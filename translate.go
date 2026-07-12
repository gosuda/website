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

func translateAndEvaluate(ctx context.Context, post *types.Post, lang types.Lang, fullLangName string, fieldName string, text string) (string, error) {
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msgf("translating post %s", fieldName)
	translatedText, err := translate.Translate(ctx, llmModel, text, fullLangName)
	if err != nil {
		return "", err
	}
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Str(fieldName, translatedText).Msgf("translated post %s", fieldName)
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Msgf("evaluating translated %s", fieldName)
	score, err := evaluate.EvaluateTranslation(ctx, llmModel, post.Main.Metadata.Language, lang, text, translatedText)
	if err != nil {
		return "", err
	}
	log.Debug().Str("path", post.FilePath).Str("lang", string(lang)).Float64("score", score).Msg("evaluated translation")
	if score < 0.7 {
		return "", ErrLowQualityTranslation
	}
	return translatedText, nil
}

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

	newTitle, err := translateAndEvaluate(ctx, post, lang, fullLangName, "title", post.Main.Metadata.Title)
	if err != nil {
		return err
	}
	meta.Title = newTitle

	newDescription, err := translateAndEvaluate(ctx, post, lang, fullLangName, "description", post.Main.Metadata.Description)
	if err != nil {
		return err
	}
	meta.Description = newDescription

	tranDocument, err := translateAndEvaluate(ctx, post, lang, fullLangName, "content", origDocument)
	if err != nil {
		return err
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
