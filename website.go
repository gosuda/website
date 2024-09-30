package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gosuda.org/website/internal/markdown"
	"gosuda.org/website/internal/types"
)

const (
	rootDir   = "root"
	publicDir = "public"
	distDir   = "dist"
)

//go:generate templ generate
//go:generate bun run build

func generateFileList(dir string) ([]string, error) {
	var fileList []string
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return fileList, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = srcFile.WriteTo(dstFile)
	if err != nil {
		return err
	}
	return nil
}

func copyDir(src, dst string) error {
	filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath := strings.TrimPrefix(path, src)
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			err := os.MkdirAll(dstPath, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			err := copyFile(path, dstPath)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}

// readMarkdownFile reads the contents of a markdown file.
func readMarkdownFile(path string) ([]byte, error) {
	log.Debug().Str("path", path).Msgf("start reading markdown file %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("path", path).Int("size", len(data)).Msgf("read markdown file %s", path)
	return data, nil
}

// parseMarkdown renders the given markdown data into HTML.
func parseMarkdown(path string, data []byte) (*types.Document, error) {
	log.Debug().Str("path", path).Msgf("rendering markdown file %s", path)
	doc, err := markdown.ParseMarkdown(string(data))
	if err != nil {
		return nil, err
	}
	log.Debug().Str("path", path).Int("rendered_size", len(doc.HTML)).Msgf("rendered markdown file %s", path)
	return doc, nil
}

// processMarkdownFile processes a markdown file and returns the rendered HTML document.
func processMarkdownFile(path string) (*types.Document, error) {
	log.Debug().Str("path", path).Msgf("start processing markdown file %s", path)
	data, err := readMarkdownFile(path)
	if err != nil {
		return nil, err
	}

	doc, err := parseMarkdown(path, data)
	if err != nil {
		return nil, err
	}

	if doc.Metadata.ID == "" {
		doc.Metadata.ID = types.RandID()
	}

	log.Debug().Str("path", path).Msgf("end processing markdown file %s", path)
	return doc, nil
}

func generate() error {
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
			_, err := processMarkdownFile(path)
			if err != nil {
				return err
			}
		default:
			log.Debug().Str("path", path).Msgf("skipping %s", path)
		}
	}

	log.Debug().Msg("end generating website")
	return nil
}

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"})

	generate()
}
