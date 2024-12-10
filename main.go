package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.eu.org/envloader"
	"gosuda.org/website/internal/evaluate"
)

var _ = func() struct{} {
	envloader.LoadEnvFile(".env")
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"})
	return struct{}{}
}()

//go:generate templ generate
//go:generate bun run build

func generate_main() {
	ds, err := initializeDatabase(dbFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize database file %s", dbFile)
	}

	gc := GenerationContext{
		DataStore: ds,
		UsedPosts: make(map[string]struct{}),
		PathMap:   make(map[string]string),
	}

	err = generate(&gc)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to generate website")
	}

	err = updateDatabase(dbFile, ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to update database file %s", dbFile)
	}

	log.Info().Msgf("website generated")
}

func remove_lang_main() {
	ds, err := initializeDatabase(dbFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize database file %s", dbFile)
	}

	post_id := os.Args[2]
	delete(ds.Posts[post_id].Translated, os.Args[3])

	err = updateDatabase(dbFile, ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to update database file %s", dbFile)
	}
}

func get_translation_main() {
	ds, err := initializeDatabase(dbFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize database file %s", dbFile)
	}

	fmt.Println(ds.Posts[os.Args[2]].Translated[os.Args[3]].Markdown)

	err = updateDatabase(dbFile, ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to update database file %s", dbFile)
	}
}
func eval_translation_main() {
	ds, err := initializeDatabase(dbFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize database file %s", dbFile)
	}

	evaluate.DEBUG_MODE = true
	orig := ds.Posts[os.Args[2]].Main
	trans := ds.Posts[os.Args[2]].Translated[os.Args[3]]
	score, err := evaluate.EvaluateTranslation(context.Background(), llmModel, orig.Metadata.Language, trans.Metadata.Language, orig.Markdown, trans.Markdown)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to evaluate translation")
	}
	fmt.Println("score:", score)

	err = updateDatabase(dbFile, ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to update database file %s", dbFile)
	}
}

func eval_all_main() {
	ds, err := initializeDatabase(dbFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize database file %s", dbFile)
	}

	for _, post := range ds.Posts {
		for lang, trans := range post.Translated {
			if lang == post.Main.Metadata.Language {
				continue
			}

			orig := post.Main
		retry:
			score, err := evaluate.EvaluateTranslation(context.Background(), llmModel, orig.Metadata.Language, trans.Metadata.Language, orig.Markdown, trans.Markdown)
			if err != nil {
				log.Error().Err(err).Msgf("failed to evaluate translation")
				goto retry
			}
			log.Info().Str("post_id", post.ID).Str("lang", lang).Float64("score", score).Msgf("translation score")

			if score < 0.7 {
				delete(post.Translated, lang)
			}
		}
	}

	err = updateDatabase(dbFile, ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to update database file %s", dbFile)
	}
}

func edit_db_main() {
	ds, err := initializeDatabase(dbFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize database file %s", dbFile)
	}

	// write to tmp file
	tmpFile, err := os.Create(dbFile + ".edit")
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to create tmp file")
	}
	defer os.Remove(dbFile + ".edit")

	e := json.NewEncoder(tmpFile)
	e.SetIndent("", "  ")
	err = e.Encode(ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to encode database")
	}

	err = tmpFile.Close()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to close tmp file")
	}

	log.Info().Msgf("database edit mode enabled, press enter to save and exit")
	fmt.Scanln()

	f, err := os.Open(dbFile + ".edit")
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to close edit file")
	}
	log.Info().Msgf("database edit mode disabled")

	err = json.NewDecoder(f).Decode(ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to decode database")
	}

	err = updateDatabase(dbFile, ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to update database file %s", dbFile)
	}
	log.Info().Msgf("database updated")
}

func main() {
	if llmClient != nil {
		defer llmClient.Close()
	}
	if llmModel != nil {
		defer llmModel.Close()
	}

	if len(os.Args) == 1 {
		generate_main()
		return
	}

	switch os.Args[1] {
	case "remove_lang":
		remove_lang_main() // remove lang from db
		return
	case "get_translation":
		get_translation_main() // get translation from db
		return
	case "eval_translation":
		eval_translation_main() // eval translation
		return
	case "eval_all":
		eval_all_main() // eval all translations and remove if it is low quality.
	case "edit_db":
		edit_db_main() // edit db
	}
}
