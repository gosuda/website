package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.eu.org/envloader"
)

var _ = func() struct{} {
	envloader.LoadEnvFile(".env")
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"})
	return struct{}{}
}()

//go:generate templ generate
//go:generate bun run build

func main() {
	if llmClient != nil {
		defer llmClient.Close()
	}
	if llmModel != nil {
		defer llmModel.Close()
	}

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

	// print database as JSON
	jsonData, err := json.MarshalIndent(&gc, "", "  ")
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to marshal database file %s", dbFile)
	}
	fmt.Println(string(jsonData))
}
