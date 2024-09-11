package markdown

import (
	"bytes"
	"errors"
	"time"

	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"mvdan.cc/xurls/v2"
)

type Document struct {
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`

	Content string `json:"content"`
	HTML    string `json:"html"`

	Path string `json:"path"`
}

var ErrInvalidMetadata = errors.New("invalid metadata")

func ParseDocument(text string) (*Document, error) {
	gMark := goldmark.New(
		goldmark.WithExtensions(
			meta.New(
				meta.WithStoresInDocument(),
			),
			extension.NewLinkify(
				extension.WithLinkifyAllowedProtocols([]string{
					"http:",
					"https:",
				}),
				extension.WithLinkifyURLRegexp(
					xurls.Strict(),
				),
			),
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
				highlighting.WithFormatOptions(
					chtml.WithLineNumbers(true),
					// chtml.WithLinkableLineNumbers(true, "CL"), I don't need this (It hurts SEO)
				),
				highlighting.WithGuessLanguage(true),
			),
			extension.GFM,
			extension.CJK,
		),
	)

	doc := Document{}
	context := parser.NewContext()

	var buf bytes.Buffer
	if err := gMark.Convert([]byte(text), &buf, parser.WithContext(context)); err != nil {
		return nil, err
	}
	metadata := meta.Get(context)

	for key, value := range metadata {
		switch key {
		case "title":
			s, ok := value.(string)
			if !ok {
				return nil, ErrInvalidMetadata
			}
			doc.Title = s
		case "author":
			s, ok := value.(string)
			if !ok {
				return nil, ErrInvalidMetadata
			}
			doc.Author = s
		case "description":
			s, ok := value.(string)
			if !ok {
				return nil, ErrInvalidMetadata
			}
			doc.Description = s
		case "date":
			s, ok := value.(time.Time)
			if !ok {
				return nil, ErrInvalidMetadata
			}
			doc.Date = s
		}
	}
	doc.Content = text
	doc.HTML = buf.String()

	return &doc, nil
}
