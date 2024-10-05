package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.eu.org/envloader"
	"gosuda.org/website/internal/types"
)

var _ = func() struct{} {
	envloader.LoadEnvFile(".env")
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"})
	return struct{}{}
}()

func main() {
	defer llmClient.Close()
	defer llmModel.Close()

	_, err := os.Stat(dbFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal().Err(err).Msgf("failed to stat database file %s", dbFile)
	}

	var f *os.File
	if err != nil && os.IsNotExist(err) {
		log.Info().Err(err).Msgf("database file %s does not exist, Creating new database file", dbFile)
		f, err = os.OpenFile(dbFile, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to create database file %s", dbFile)
		}

		w, err := zstd.NewWriter(f)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to create zstd writer for database file %s", dbFile)
		}

		_, err = w.Write([]byte("{}"))
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to write to database file %s", dbFile)
		}

		err = w.Close()
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to close zstd writer for database file %s", dbFile)
		}

		_, err = f.Seek(0, 0)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to seek to beginning of database file %s", dbFile)
		}
	} else {
		f, err = os.OpenFile(dbFile, os.O_RDWR, 0644)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to open database file %s", dbFile)
		}
	}

	var gc GenerationContext
	var ds DataStore
	gc.DataStore = &ds

	r, err := zstd.NewReader(f)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to create zstd reader for database file %s", dbFile)
	}

	err = json.NewDecoder(r).Decode(&ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to decode database file %s", dbFile)
	}
	r.Close()

	err = f.Close()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to close database file %s", dbFile)
	}

	if gc.DataStore == nil {
		gc.DataStore = &DataStore{}
	}

	if gc.DataStore.Posts == nil {
		gc.DataStore.Posts = make(map[string]*types.Post)
	}

	err = generate(&gc)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to generate website")
	}

	// Update Database
	f, err = os.OpenFile(dbFile+".tmp", os.O_CREATE|os.O_RDWR|os.O_TRUNC|os.O_EXCL, 0644)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to create temporary database file %s", dbFile)
	}

	w, err := zstd.NewWriter(f, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to create zstd writer for database file %s", dbFile)
	}

	err = json.NewEncoder(w).Encode(&ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to encode database file %s", dbFile)
	}

	err = w.Close()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to close zstd writer for database file %s", dbFile)
	}

	err = f.Close()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to close database file %s", dbFile)
	}

	err = os.Rename(dbFile+".tmp", dbFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to rename temporary database file %s", dbFile)
	}

	log.Info().Msgf("database file %s updated", dbFile)
	log.Info().Msgf("website generated")

	// print database as JSON
	jsonData, err := json.MarshalIndent(&gc, "", "  ")
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to marshal database file %s", dbFile)
	}
	fmt.Println(string(jsonData))
}
