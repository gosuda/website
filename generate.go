package main

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gosuda.org/website/internal/ogimage"
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

	// Remove unused posts
	for id := range gc.DataStore.Posts {
		if _, ok := gc.UsedPosts[id]; !ok {
			log.Debug().Str("id", id).Msgf("removing unused post %s", id)
			delete(gc.DataStore.Posts, id)
		}
	}

	for _, lang := range types.SupportedLanguages {
		err = generateIndex(gc, lang)
		if err != nil {
			return err
		}
		err = generatePostPages(gc, lang)
		if err != nil {
			return err
		}
	}

	err = minifyDir(distDir)
	if err != nil {
		return err
	}

	log.Debug().Msg("done generating website")
	return nil
}

func generatePostPages(gc *GenerationContext, lang types.Lang) error {
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
		pm := post.Main.Metadata
		if lang != pm.Language {
			if _, ok := post.Translated[lang]; ok {
				pm = post.Translated[lang].Metadata
			} else {
				continue
			}
		}

		path := post.Path
		if lang != types.LangEnglish {
			path = "/" + lang + path
		}

		log.Debug().Str("path", post.Path).Msgf("generating post page %s", path)

		fp := filepath.Join(distDir, path)
		err := os.MkdirAll(filepath.Dir(fp), 0755)
		if err != nil {
			return err
		}

		if lang == types.LangEnglish {
			err := os.MkdirAll(filepath.Dir(filepath.Join(distDir, post.Path)), 0755)
			if err != nil {
				return err
			}
		}

		ogImagePath := filepath.Join(distDir, "assets", post.ID+"_"+lang+".png")
		err = os.MkdirAll(filepath.Dir(ogImagePath), 0755)
		if err != nil {
			return err
		}

		url := baseURL + "/" + lang + post.Path
		if lang == types.LangEnglish {
			url = baseURL + post.Path
		}

		meta := &view.Metadata{
			Language:    lang,
			Title:       pm.Title,
			Description: pm.Description,
			Author:      pm.Author,
			Image:       baseURL + "/assets/" + post.ID + "_" + lang + ".png",
			URL:         url,
			Canonical:   url,
			BaseURL:     baseURL,
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
			err = os.WriteFile(fp+"index.html", b.Bytes(), 0644)
			if err != nil {
				return err
			}
		} else {
			err = os.WriteFile(fp+".html", b.Bytes(), 0644)
			if err != nil {
				return err
			}
		}

		if lang == types.LangEnglish {
			fp = filepath.Join(distDir, post.Path)
			if strings.HasSuffix(fp, "/") {
				err = os.WriteFile(fp+"index.html", b.Bytes(), 0644)
				if err != nil {
					return err
				}
			} else {
				err = os.WriteFile(fp+".html", b.Bytes(), 0644)
				if err != nil {
					return err
				}
			}
		}

		img := ogimage.GenerateImage("GoSuda", pm.Title, pm.Date)
		f, err := os.Create(ogImagePath)
		if err != nil {
			return err
		}

		err = png.Encode(f, img)
		if err != nil {
			err = f.Close()
			if err != nil {
				return err
			}
			return err
		}

		err = f.Close()
		if err != nil {
			return err
		}
	}

	log.Debug().Msg("done generating post pages")
	return nil
}

func generateIndex(gc *GenerationContext, lang types.Lang) error {
	log.Debug().Msg("start generating index")
	var b bytes.Buffer
	ctx := context.Background()

	err := os.MkdirAll(filepath.Join(distDir, lang), 0755)
	if err != nil {
		return err
	}

	meta := &view.Metadata{
		Language:    lang,
		Title:       "GoSuda | Home",
		Description: "GoSuda is an industry-leading open source working group enabling developers to easily build, prototype, and deploy applications. Our comprehensive suite of tools and frameworks empowers developers to create robust, scalable solutions across various domains.",
		Author:      "GoSuda",
		Image:       baseURL + "/assets/images/ogp_placeholder.png",
		URL:         baseURL + "/",
		Canonical:   baseURL + "/",
		BaseURL:     baseURL,
		CreatedAt:   time.Date(2024, 10, 07, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Now().UTC(),
	}

	if lang != "en" {
		meta.URL = baseURL + "/" + lang + "/"
		meta.Canonical = baseURL + "/" + lang + "/"
	}

	var posts []*types.Post
	for _, post := range gc.DataStore.Posts {
		if post.Main.Metadata.Hidden {
			continue
		}
		posts = append(posts, post)
	}
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Main.Metadata.Date.After(posts[j].Main.Metadata.Date)
	})

	if len(posts) > 16 {
		posts = posts[:16]
	}

	var previews []*view.BlogPostPreview
	for _, post := range posts {
		pm := post.Main.Metadata
		if lang != pm.Language {
			if _, ok := post.Translated[lang]; ok {
				pm = post.Translated[lang].Metadata
			} else {
				continue
			}
		}

		postPath := post.Path

		if lang != "en" {
			postPath = "/" + lang + post.Path
		}

		previews = append(previews, &view.BlogPostPreview{
			Title:       pm.Title,
			Author:      pm.Author,
			Description: pm.Description,
			Date:        pm.Date,
			URL:         postPath,
		})
	}

	var featuredPosts []view.FeaturedPost

	err = view.IndexPage(meta, previews, featuredPosts).Render(ctx, &b)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, lang, "index.html"), b.Bytes(), 0644)
	if err != nil {
		return err
	}

	if lang == "en" {
		err = os.WriteFile(filepath.Join(distDir, "index.html"), b.Bytes(), 0644)
		if err != nil {
			return err
		}
	}

	log.Debug().Msg("done generating index")
	return nil
}
