package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/feeds"
	"github.com/rs/zerolog/log"
	"github.com/zeebo/blake3"
	"gosuda.org/website/internal/types"
	"gosuda.org/website/view"
)

func langFeedID(id string, lang types.Lang) string {
	var buf [16]byte
	blake3.DeriveKey("LANGUAGE FEED ID v0.1 LANG:"+lang, []byte(id), buf[:])
	return hex.EncodeToString(buf[:])
}

func createFeedItem(post *types.Post, doc *types.Document, link string) *feeds.Item {
	return &feeds.Item{
		Id:          langFeedID(post.ID, doc.Metadata.Language),
		Title:       doc.Metadata.Title,
		Link:        &feeds.Link{Href: link},
		Author:      &feeds.Author{Name: doc.Metadata.Author},
		Description: doc.Metadata.Description,
		Created:     post.CreatedAt.UTC(),
		Updated:     post.UpdatedAt.UTC(),
	}
}

func generateGlobalFeed(gc *GenerationContext) error {
	log.Debug().Msg("start generating global RSS feed")
	globalFeed := &feeds.Feed{
		Title:       "Gosuda Blog",
		Link:        &feeds.Link{Href: "https://gosuda.org/"},
		Description: "Gosuda: A blog about software development, and other topics.",
		Author:      &feeds.Author{Name: "Gosuda", Email: "webmaster@gosuda.org"},
		Created:     time.Now().UTC(),
	}

	for _, post := range gc.DataStore.Posts {
		doc := post.Main
		if doc.Metadata.Language != "en" {
			enDoc, ok := post.Translated["en"]
			if !ok {
				continue
			}
			doc = enDoc
		}
		link := baseURL + post.Path

		globalFeed.Items = append(globalFeed.Items, createFeedItem(post, doc, link))
	}

	globalFeed.Items = append(globalFeed.Items, &feeds.Item{
		Id:          langFeedID("home", types.LangEnglish),
		Title:       "GoSuda | Home",
		Link:        &feeds.Link{Href: baseURL + "/"},
		Author:      &feeds.Author{Name: "GoSuda"},
		Description: "GoSuda is an industry-leading open source working group enabling developers to easily build, prototype, and deploy applications. Our comprehensive suite of tools and frameworks empowers developers to create robust, scalable solutions across various domains.",
		Created:     time.Date(2024, 10, 07, 0, 0, 0, 0, time.UTC),
		Updated:     time.Now().UTC(),
	})

	rss, err := globalFeed.ToRss()
	if err != nil {
		return err
	}

	sitemap, err := encodeSiteMapXML(globalFeed)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, "feed.rss"), []byte(rss), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, "en", "feed.rss"), []byte(rss), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, "sitemap.xml"), []byte(sitemap), 0644)
	if err != nil {
		return err
	}

	log.Debug().Msg("done generating global RSS feed")
	return nil
}

func generateLocalFeed(gc *GenerationContext, lang types.Lang) error {
	log.Debug().Str("lang", string(lang)).Msg("start generating local RSS feed")

	feed := &feeds.Feed{
		Title:       "GoSuda Blog" + " - " + types.FullLangName(lang),
		Link:        &feeds.Link{Href: baseURL + "/" + string(lang) + "/"},
		Description: "Gosuda: A blog about software development, and other topics.",
		Author:      &feeds.Author{Name: "Gosuda", Email: "webmaster@gosuda.org"},
		Created:     time.Now().UTC(),
	}

	for _, post := range gc.DataStore.Posts {
		doc, ok := post.Translated[string(lang)]
		if !ok {
			continue
		}
		link := baseURL + "/" + string(lang) + post.Path

		feed.Items = append(feed.Items, createFeedItem(post, doc, link))
	}

	feed.Items = append(feed.Items, &feeds.Item{
		Id:          langFeedID("home", lang),
		Title:       "GoSuda | Home",
		Link:        &feeds.Link{Href: baseURL + "/" + string(lang) + "/"},
		Author:      &feeds.Author{Name: "GoSuda"},
		Description: "GoSuda is an industry-leading open source working group enabling developers to easily build, prototype, and deploy applications. Our comprehensive suite of tools and frameworks empowers developers to create robust, scalable solutions across various domains.",
		Created:     time.Date(2024, 10, 07, 0, 0, 0, 0, time.UTC),
		Updated:     time.Now().UTC(),
	})

	rss, err := feed.ToRss()
	if err != nil {
		return err
	}

	sitemap, err := encodeSiteMapXML(feed)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, string(lang), "feed.rss"), []byte(rss), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(distDir, string(lang), "sitemap.xml"), sitemap, 0644)
	if err != nil {
		return err
	}

	log.Debug().Str("lang", string(lang)).Msg("done generating local RSS feed")
	return nil
}

const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"

func encodeSiteMapXML(feed *feeds.Feed) ([]byte, error) {
	var b bytes.Buffer
	err := view.Sitemap(feed).Render(context.Background(), &b)
	if err != nil {
		return nil, err
	}
	return append([]byte(xmlHeader), b.Bytes()...), nil
}
