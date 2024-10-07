package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	mjson "github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

var minifier = minify.New()

func init() {
	minifier.Add("text/html", &html.Minifier{
		KeepDocumentTags: true,
		KeepEndTags:      true,
	})
	minifier.AddFunc("text/css", css.Minify)
	minifier.AddFunc("image/svg+xml", svg.Minify)
	minifier.AddFunc("application/javascript", js.Minify)
	minifier.AddFunc("application/json", mjson.Minify)
	minifier.AddFunc("application/xml", xml.Minify)
}

func minifyFile(path string, mime string) error {
	log.Debug().Str("path", path).Msgf("minifying file %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	data, err = minifier.Bytes(mime, data)
	if err != nil {
		return err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, stat.Mode())
	if err != nil {
		return err
	}
	log.Debug().Str("path", path).Msgf("minified file %s", path)
	return nil
}

func minifyDir(dir string) error {
	list, err := generateFileList(dir)
	if err != nil {
		return err
	}

	for _, path := range list {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".html", ".htm":
			err = minifyFile(path, "text/html")
		case ".css":
			err = minifyFile(path, "text/css")
		case ".js":
			err = minifyFile(path, "application/javascript")
		case ".svg":
			err = minifyFile(path, "image/svg+xml")
		case ".json":
			err = minifyFile(path, "application/json")
		case ".xml":
			err = minifyFile(path, "application/xml")
		default:
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}
