package types

import "time"

type Post struct {
	ID   string `json:"id"`
	Path string `json:"path"`

	GoPackage string `json:"go_package,omitempty"`
	Canonical string `json:"canonical"`
	Hidden    bool   `json:"hidden"`

	Main       *Document            `json:"main"`
	Translated map[string]*Document `json:"translated"`
}

//go:generate stringer -type=DocumentType -output=post_types.go -linecomment
type DocumentType uint16

const (
	DocumentTypeUnknown  DocumentType = iota // unknown
	DocumentTypeMarkdown                     // markdown
	DocumentTypeHTML                         // html
)

func (g DocumentType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + g.String() + `"`), nil
}

type Document struct {
	Type DocumentType `json:"type"`

	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`

	// The content of the Markdown document.
	Markdown string `json:"markdown"`
	// The content of the HTML document or the rendered HTML of the Markdown document.
	HTML string `json:"html"`
	// Parsed metadata from the Markdown document.
	Metadata map[string]interface{} `json:"metadata"`
}

func GetPosts()
