package main

import (
	"encoding/json"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog/log"
	"gosuda.org/website/internal/types"
)

func initializeDatabase(dbFile string) (*DataStore, error) {
	_, err := os.Stat(dbFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var f *os.File
	if err != nil && os.IsNotExist(err) {
		log.Info().Err(err).Msgf("database file %s does not exist, Creating new database file", dbFile)
		f, err = os.OpenFile(dbFile, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			return nil, err
		}

		w, err := zstd.NewWriter(f)
		if err != nil {
			return nil, err
		}

		_, err = w.Write([]byte("{}"))
		if err != nil {
			return nil, err
		}

		err = w.Close()
		if err != nil {
			return nil, err
		}

		_, err = f.Seek(0, 0)
		if err != nil {
			return nil, err
		}
	} else {
		f, err = os.OpenFile(dbFile, os.O_RDWR, 0644)
		if err != nil {
			return nil, err
		}
	}
	defer f.Close()

	var ds DataStore

	r, err := zstd.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	err = json.NewDecoder(r).Decode(&ds)
	if err != nil {
		return nil, err
	}

	if ds.Posts == nil {
		ds.Posts = make(map[string]*types.Post)
	}

	return &ds, nil
}

func updateDatabase(dbFile string, ds *DataStore) error {
	f, err := os.OpenFile(dbFile+".tmp", os.O_CREATE|os.O_RDWR|os.O_TRUNC|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := zstd.NewWriter(f, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return err
	}
	defer w.Close()

	err = json.NewEncoder(w).Encode(ds)
	if err != nil {
		return err
	}

	err = os.Rename(dbFile+".tmp", dbFile)
	if err != nil {
		return err
	}

	log.Info().Msgf("database file %s updated", dbFile)
	return nil
}
