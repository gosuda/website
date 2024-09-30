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
	"gosuda.org/website/internal/types"
	"mvdan.cc/xurls/v2"
)

var ErrInvalidMetadata = errors.New("invalid metadata")

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
)

func parseMetadata(doc *types.Document, metadata map[string]interface{}) error {
	m := &doc.Metadata
	for key, value := range metadata {
		switch key {
		case "id":
			if s, ok := value.(string); ok {
				m.ID = s
			}
		case "title":
			if s, ok := value.(string); ok {
				m.Title = s
			}
		case "author":
			if s, ok := value.(string); ok {
				m.Author = s
			}
		case "description":
			if s, ok := value.(string); ok {
				m.Description = s
			}
		case "date":
			if t, ok := value.(time.Time); ok {
				m.Date = t
			}
		case "go_package":
			if s, ok := value.(string); ok {
				m.GoPackage = s
			}
		case "canonical":
			if s, ok := value.(string); ok {
				m.Canonical = s
			}
		case "hidden":
			if b, ok := value.(bool); ok {
				m.Hidden = b
			}
		}
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
