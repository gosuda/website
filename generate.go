package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"gosuda.org/website/internal/types"
	"gosuda.org/website/view"
)

func generate(gc *GenerationContext) error {
	log.Debug().Msg("start generating website")

	distInfo, err := os.Stat(distDir)
	if err == nil && distInfo.IsDir() {
		log.Debug().Msg("deleting dist directory")
		err := os.RemoveAll(distDir)
		if err != nil {
			return err
		}
		log.Debug().Msg("deleted dist directory")
	}

	log.Debug().Msg("copying static files")
	err = copyDir(publicDir, distDir)
	if err != nil {
		return err
	}
	log.Debug().Msg("copied static files")

	log.Debug().Msg("creating root file index")
	list, err := generateFileList(rootDir)
	if err != nil {
		return err
	}

	for _, path := range list {
		log.Debug().Str("path", path).Msgf("processing file %s", path)
		switch strings.ToLower(filepath.Ext(path)) {
		case ".md", ".markdown":
			_, err := processMarkdownFile(gc, path)
			if err != nil {
				log.Error().Err(err).Str("path", path).Msgf("failed to process markdown file %s", path)
			}
		default:
			log.Debug().Str("path", path).Msgf("skipping %s", path)
		}
		log.Debug().Str("path", path).Msgf("processed file %s", path)
	}

	err = generatePostPages(gc)
	if err != nil {
		return err
	}

	err = minifyDir(distDir)
	if err != nil {
		return err
	}

	// Remove unused posts
	for id := range gc.DataStore.Posts {
		if _, ok := gc.UsedPosts[id]; !ok {
			log.Debug().Str("id", id).Msgf("removing unused post %s", id)
			delete(gc.DataStore.Posts, id)
		}
	}

	log.Debug().Msg("done generating website")
	return nil
}

func generatePostPages(gc *GenerationContext) error {
	log.Debug().Msg("start generating post pages")
	postList := make([]*types.Post, 0, len(gc.DataStore.Posts))
	for _, post := range gc.DataStore.Posts {
		postList = append(postList, post)
	}

	sort.Slice(postList, func(i, j int) bool {
		return postList[i].ID < postList[j].ID
	})

	var b bytes.Buffer
	ctx := context.Background()

	for _, post := range postList {
		log.Debug().Str("path", post.Path).Msgf("generating post page %s", post.Path)
		fp := filepath.Join(distDir, post.Path)
		err := os.MkdirAll(filepath.Dir(fp), 0755)
		if err != nil {
			return err
		}

		meta := &view.Metadata{
			Language:    post.Main.Metadata.Language,
			Title:       post.Main.Metadata.Title,
			Description: post.Main.Metadata.Description,
			Author:      post.Main.Metadata.Author,
			URL:         baseURL + post.Path,
			BaseURL:     baseURL,
			Canonical:   baseURL + post.Path,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
		}

		if post.Main.Metadata.Canonical != "" {
			meta.Canonical = post.Main.Metadata.Canonical
		}

		if post.Main.Metadata.GoPackage != "" {
			meta.GoImport = fmt.Sprintf("%s git %s", post.Main.Metadata.GoPackage, post.Main.Metadata.GoRepoURL)
		}

		b.Reset()
		err = view.PostPage(meta, post.Main, post).Render(ctx, &b)
		if err != nil {
			return err
		}

		if strings.HasSuffix(fp, "/") {
			fp += "index.html"
		} else {
			fp += ".html"
		}

		err = os.WriteFile(fp, b.Bytes(), 0644)
		if err != nil {
			return err
		}
	}

	log.Debug().Msg("done generating post pages")
	return nil
}
