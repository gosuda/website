package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/rs/zerolog/log"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"gopkg.in/yaml.v3"
	"gosuda.org/website/internal/types"
	"mvdan.cc/xurls/v2"
)

var ErrInvalidMetadata = errors.New("invalid metadata")

type imageDimensionTransformer struct{}

var defaultImageDimensionTransformer = &imageDimensionTransformer{}

func (t *imageDimensionTransformer) Transform(node *ast.Document, reader text.Reader, pctx parser.Context) {
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if n.Kind() == ast.KindImage {
			img := n.(*ast.Image)
			src := string(img.Destination)

			log.Debug().Str("src", src).Msg("Processing image")

			// Check if the image source is a local path
			if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
				// Construct the full path to the image file.
				// Assuming image paths in markdown are relative to the 'public' directory.
				// This might need adjustment based on actual image location practices.
				imgPath := filepath.Join("public", src)

				file, err := os.Open(imgPath)
				if err != nil {
					log.Error().Err(err).Str("path", imgPath).Msg("Error opening local image file")
					return ast.WalkContinue, nil
				}
				defer file.Close()

				imgConfig, _, err := image.DecodeConfig(file)
				if err != nil {
					log.Error().Err(err).Str("path", imgPath).Msg("Error decoding local image config")
					return ast.WalkContinue, nil
				}

				// Add width and height attributes
				img.SetAttributeString("width", []byte(fmt.Sprintf("%d", imgConfig.Width)))
				img.SetAttributeString("height", []byte(fmt.Sprintf("%d", imgConfig.Height)))
				log.Debug().Str("src", src).Int("width", imgConfig.Width).Int("height", imgConfig.Height).Msg("Added dimensions to local image")
			} else {
				// Handle external images
				client := http.Client{
					Timeout: 5 * time.Second, // Set a timeout for fetching the image header
				}

				resp, err := client.Get(src)
				if err != nil {
					log.Error().Err(err).Str("src", src).Msg("Error fetching external image")
					return ast.WalkContinue, nil
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					log.Error().Str("src", src).Int("status", resp.StatusCode).Msg("External image returned non-OK status")
					return ast.WalkContinue, nil
				}

				imgConfig, _, err := image.DecodeConfig(resp.Body)
				if err != nil {
					log.Error().Err(err).Str("src", src).Msg("Error decoding external image config")
					return ast.WalkContinue, nil
				}

				// Add width and height attributes
				img.SetAttributeString("width", []byte(fmt.Sprintf("%d", imgConfig.Width)))
				img.SetAttributeString("height", []byte(fmt.Sprintf("%d", imgConfig.Height)))
				log.Debug().Str("src", src).Int("width", imgConfig.Width).Int("height", imgConfig.Height).Msg("Added dimensions to external image")
			}
		}
		return ast.WalkContinue, nil
	})
}

var gMark = goldmark.New(
	goldmark.WithExtensions(
		meta.New(meta.WithStoresInDocument()),
		extension.NewLinkify(
			extension.WithLinkifyAllowedProtocols([]string{"http:", "https:"}),
			extension.WithLinkifyURLRegexp(xurls.Strict()),
		),
		highlighting.NewHighlighting(
			highlighting.WithStyle("dracula"),
			highlighting.WithFormatOptions(
				chtml.WithLineNumbers(true),
			),
			highlighting.WithGuessLanguage(true),
		),
		extension.GFM,
		extension.CJK,
	),
	goldmark.WithParserOptions(
		parser.WithASTTransformers(
			util.Prioritized(defaultImageDimensionTransformer, 0),
		),
	),
)

func parseMetadata(doc *types.Document, metadata map[string]interface{}) error {
	m := &doc.Metadata

	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlData, m)
	if err != nil {
		return err
	}

	// If ID is not set in metadata, generate a random one
	if m.ID == "" {
		m.ID = types.RandID()
	}

	return nil
}
func ParseMarkdown(text string) (*types.Document, error) {
	doc := &types.Document{
		Type:     types.DocumentTypeMarkdown,
		Markdown: text,
	}

	context := parser.NewContext()
	var buf bytes.Buffer

	err := gMark.Convert([]byte(text), &buf, parser.WithContext(context))
	if err != nil {
		return nil, err
	}

	metadata := meta.Get(context)
	err = parseMetadata(doc, metadata)
	if err != nil {
		return nil, err
	}

	doc.HTML = buf.String()

	return doc, nil
}
