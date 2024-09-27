package types

import (
	"time"
)

// Post represents a blog post or similar content item.
type Post struct {
	// ID is the unique identifier for the post.
	ID string `json:"id"`
	// Path is the URL path or slug for the post.
	Path string `json:"path"`

	// GoPackage is the Go package associated with the post (optional).  Likely used if the post is generated from Go source.
	GoPackage string `json:"go_package,omitempty"`
	// Canonical is the canonical URL for the post.
	Canonical string `json:"canonical,omitempty"`
	// Hidden indicates whether the post should be listed on the front page.
	Hidden bool `json:"hidden"`

	// Main contains the primary language version of the post content.
	Main *Document `json:"main"`
	// Translated contains translated versions of the post content, keyed by language code.
	Translated map[string]*Document `json:"translated"`
}

// DocumentType represents the type of a document (e.g., Markdown, HTML).
//
//go:generate stringer -type=DocumentType -output=post_types.go -linecomment
type DocumentType uint16

const (
	DocumentTypeUnknown  DocumentType = iota // unknown
	DocumentTypeMarkdown                     // markdown
	DocumentTypeHTML                         // html
)

// MarshalJSON customizes the JSON marshaling of DocumentType to output the string representation.
func (g DocumentType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + g.String() + `"`), nil
}

// Document represents the content and metadata of a post in a specific language.
type Document struct {
	// Type indicates the format of the document content.
	Type DocumentType `json:"type"`

	// Title is the title of the document.
	Title string `json:"title"`
	// Author is the author of the document.
	Author string `json:"author"`
	// Description is a brief description of the document.
	Description string `json:"description"`
	// Date is the publication date of the document.
	Date time.Time `json:"date"`

	// Markdown is the raw Markdown content of the document (if applicable).
	Markdown string `json:"markdown"`
	// HTML is the HTML content of the document, either directly provided or rendered from Markdown.
	HTML string `json:"html"`
	// Metadata contains any additional metadata parsed from the Markdown document.
	Metadata map[string]interface{} `json:"metadata"`
}
