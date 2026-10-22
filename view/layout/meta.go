package layout

import "time"

type Metadata struct {
	Language    string
	Title       string
	Description string
	Author      string
	Keywords    []string
	Image       string
	URL         string
	BaseURL     string
	Canonical   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	GoImport    string
	CustomHead  string

	Alternate *Alternate
}

type Alternate struct {
	Default  string
	Versions []KV
}

type KV struct {
	Key   string
	Value string
}

// HomeURL returns the localized home page path.
func (m *Metadata) HomeURL() string {
	if m.Language == "en" {
		return "/"
	}
	return "/" + m.Language + "/"
}
