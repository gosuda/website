package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"gosuda.org/website/internal/evaluate"
	"gosuda.org/website/internal/markdown"
	"gosuda.org/website/internal/translate"
	"gosuda.org/website/internal/types"
)

const translationConcurrency = 16

var postTranslatedMu sync.Mutex

func collectPostTranslationLangs(post *types.Post, retranslate bool, ignoreLangs ...types.Lang) []types.Lang {
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

	var langs []types.Lang
	if !retranslate {
		// only retranslate the missing languages
		for _, lang := range types.SupportedLanguages {
			if _, ok := post.Translated[string(lang)]; !ok && !slices.Contains(ignoreLangs, lang) {
				langs = append(langs, lang)
			}
		}
	} else {
		// all supported languages except the ones in ignoreLangs
		for _, lang := range types.SupportedLanguages {
			if !slices.Contains(ignoreLangs, lang) {
				delete(post.Translated, string(lang))
				langs = append(langs, lang)
			}
		}
	}

	return langs
}

func enqueuePostTranslations(gc *GenerationContext, post *types.Post, retranslate bool, ignoreLangs ...types.Lang) {
	langs := collectPostTranslationLangs(post, retranslate, ignoreLangs...)
	for _, lang := range langs {
		gc.TranslationJobs = append(gc.TranslationJobs, TranslationJob{
			Post: post,
			Lang: lang,
		})
	}
}

func runTranslationJobs(gc *GenerationContext) {
	executeTranslationJobs(context.Background(), gc.TranslationJobs)
}

func executeTranslationJobs(ctx context.Context, jobs []TranslationJob) {
	total := len(jobs)
	if total == 0 {
		return
	}

	type scheduledJob struct {
		job       TranslationJob
		followUps []TranslationJob
	}

	var firstJobs []scheduledJob
	indexByPost := make(map[*types.Post]int)
	for _, job := range jobs {
		if idx, exists := indexByPost[job.Post]; exists {
			firstJobs[idx].followUps = append(firstJobs[idx].followUps, job)
		} else {
			indexByPost[job.Post] = len(firstJobs)
			firstJobs = append(firstJobs, scheduledJob{job: job})
		}
	}

	workers := min(translationConcurrency, total)
	log.Info().Int("jobs", total).Int("posts", len(firstJobs)).Int("workers", workers).Msg("starting parallel translation")

	jobCh := make(chan scheduledJob, total)
	for _, sj := range firstJobs {
		jobCh <- sj
	}

	var completed atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for sj := range jobCh {
				job := sj.job
				err := translateWithRetry(ctx, job.Post, job.Lang)
				for _, next := range sj.followUps {
					jobCh <- scheduledJob{job: next}
				}
				done := completed.Add(1)
				if done == int64(total) {
					close(jobCh)
				}
				if err != nil {
					log.Error().
						Err(err).
						Int64("completed", done).
						Int("total", total).
						Str("path", job.Post.FilePath).
						Str("lang", string(job.Lang)).
						Msg("translation job failed after retries")
					continue
				}
				log.Info().
					Int64("completed", done).
					Int("total", total).
					Str("path", job.Post.FilePath).
					Str("lang", string(job.Lang)).
					Msg("translation job completed")
			}
		}()
	}
	wg.Wait()
}

func translateWithRetry(ctx context.Context, post *types.Post, lang types.Lang) error {
	var lastErr error
	for retry := 1; retry <= 3; retry++ {
		if retry > 1 {
			log.Debug().Int("retry", retry).Str("path", post.FilePath).Str("lang", string(lang)).Msg("retrying translation")
			time.Sleep(time.Second * 3)
		}

		err := translateLang(ctx, post, lang)
		if err != nil {
			lastErr = err
			log.Error().Err(err).Int("retry", retry).Str("path", post.FilePath).Str("lang", string(lang)).Msg("failed to translate, retrying")
			continue
		}
		return nil
	}
	return lastErr
}

func translatePost(_ *GenerationContext, post *types.Post, retranslate bool, ignoreLangs ...types.Lang) error {
	langs := collectPostTranslationLangs(post, retranslate, ignoreLangs...)
	if len(langs) == 0 {
		return nil
	}
	jobs := make([]TranslationJob, 0, len(langs))
	for _, lang := range langs {
		jobs = append(jobs, TranslationJob{Post: post, Lang: lang})
	}
	executeTranslationJobs(context.Background(), jobs)
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
	postTranslatedMu.Lock()
	post.Translated[string(lang)] = doc
	postTranslatedMu.Unlock()

	return nil
}

type RetranslateTarget struct {
	Path   string
	ID     string
	Reason string
	Langs  int
}

func normalizeDocumentForHash(doc *types.Document) error {
	if doc.Type != types.DocumentTypeMarkdown {
		return nil
	}
	if doc.Metadata.Language == "" {
		detectedLang, ok := languageDetector.DetectLanguageOf(doc.Markdown)
		if ok {
			doc.Metadata.Language = mapDetectedLanguage(detectedLang)
		} else {
			doc.Metadata.Language = "en"
		}
	}

	newMeta, err := yaml.Marshal(&doc.Metadata)
	if err != nil {
		return err
	}

	original := strings.TrimPrefix(doc.Markdown, "---\n")
	_, origDocument, ok := strings.Cut(original, "---\n")
	if !ok {
		return ErrInvalidMarkdown
	}
	doc.Markdown = "---\n" + string(newMeta) + "---\n" + origDocument
	return nil
}

func checkRetranslateTargets(ds *DataStore, dir string) ([]RetranslateTarget, error) {
	list, err := generateFileList(dir)
	if err != nil {
		return nil, err
	}

	var targets []RetranslateTarget
	for _, path := range list {
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		data = []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))

		doc, err := parseMarkdown(path, data)
		if err != nil {
			return nil, err
		}

		if doc.Metadata.NoTranslate {
			continue
		}

		hasMissingRequiredMeta := doc.Metadata.ID == "" || doc.Metadata.Date.IsZero() || doc.Metadata.Path == "" || doc.Metadata.Description == ""
		if err := normalizeDocumentForHash(doc); err != nil {
			return nil, err
		}

		ignoreLangs := append([]string{doc.Metadata.Language}, doc.Metadata.IgnoreLangs...)
		fullTargetCount := 0
		for _, lang := range types.SupportedLanguages {
			if !slices.Contains(ignoreLangs, lang) {
				fullTargetCount++
			}
		}
		if fullTargetCount == 0 {
			continue
		}

		post, ok := ds.Posts[doc.Metadata.ID]
		if !ok || hasMissingRequiredMeta {
			targets = append(targets, RetranslateTarget{
				Path:   path,
				ID:     doc.Metadata.ID,
				Reason: "new_post",
				Langs:  fullTargetCount,
			})
			continue
		}

		if post.Hash != doc.Hash() {
			targets = append(targets, RetranslateTarget{
				Path:   path,
				ID:     doc.Metadata.ID,
				Reason: "hash_changed",
				Langs:  fullTargetCount,
			})
			continue
		}

		missingCount := 0
		for _, lang := range types.SupportedLanguages {
			if _, translated := post.Translated[string(lang)]; !translated && !slices.Contains(ignoreLangs, lang) {
				missingCount++
			}
		}
		if missingCount > 0 {
			targets = append(targets, RetranslateTarget{
				Path:   path,
				ID:     doc.Metadata.ID,
				Reason: "missing_langs",
				Langs:  missingCount,
			})
		}
	}

	return targets, nil
}
