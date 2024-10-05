package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	_ "github.com/lemon-mint/coord/provider/aistudio"
	_ "github.com/lemon-mint/coord/provider/anthropic"
	_ "github.com/lemon-mint/coord/provider/openai"
	_ "github.com/lemon-mint/coord/provider/vertexai"
	"gopkg.eu.org/envloader"

	"github.com/klauspost/compress/zstd"
	"github.com/pemistahl/lingua-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	mjson "github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
	"gopkg.in/yaml.v3"

	"gosuda.org/website/internal/description"
	"gosuda.org/website/internal/markdown"
	"gosuda.org/website/internal/types"
	"gosuda.org/website/view"
)

var _ = func() struct{} {
	envloader.LoadEnvFile(".env")
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"})
	return struct{}{}
}()

const (
	rootDir   = "root"
	publicDir = "public"
	distDir   = "dist"
	dbFile    = "zdata/data.json.zstd"
	baseURL   = "https://gosuda.org"
)

var (
	ErrInvalidMarkdown = fmt.Errorf("invalid markdown file")
)

var minifier = minify.New()

func init() {
	minifier.AddFunc("text/html", html.Minify)
	minifier.AddFunc("text/css", css.Minify)
	minifier.AddFunc("image/svg+xml", svg.Minify)
	minifier.AddFunc("application/javascript", js.Minify)
	minifier.AddFunc("application/json", mjson.Minify)
	minifier.AddFunc("application/xml", xml.Minify)
}

var languageDetector lingua.LanguageDetector

func init() {
	languages := []lingua.Language{
		lingua.English,
		lingua.Spanish,
		lingua.Chinese,
		lingua.Korean,
		lingua.Japanese,
		lingua.German,
		lingua.Russian,
		lingua.French,
		lingua.Dutch,
		lingua.Italian,
		lingua.Indonesian,
		lingua.Portuguese,
		lingua.Swedish,
	}

	languageDetector = lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		Build()
}

var llmClient provider.LLMClient
var llmModel llm.Model

func Ptr[T any](t T) *T {
	return &t
}

func init() {
	log.Debug().Str("location", os.Getenv("LOCATION")).Str("project_id", os.Getenv("PROJECT_ID")).Msg("initializing llm client")
	client, err := coord.NewLLMClient(
		context.Background(),
		"vertexai",
		pconf.WithLocation(os.Getenv("LOCATION")),
		pconf.WithProjectID(os.Getenv("PROJECT_ID")),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create llm client")
	}
	llmClient = client
	log.Debug().Err(err).Msg("llm client initialized")

	llmModel, err = llmClient.NewLLM("gemini-pro-experimental", &llm.Config{
		Temperature:           Ptr(float32(0.7)),
		MaxOutputTokens:       Ptr(8192),
		SafetyFilterThreshold: llm.BlockOnlyHigh,
	})
}

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

	sort.Strings(fileList)
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

func generatePath(title string) string {
	fp := strings.TrimPrefix(title, rootDir)
	for strings.HasPrefix(fp, "/") {
		fp = strings.TrimPrefix(fp, "/")
	}

	fp = strings.ToLower(fp)
	fp = strings.ReplaceAll(fp, " ", "-")
	fp = strings.ReplaceAll(fp, "/", "-")
	fp = strings.ReplaceAll(fp, `{`, "-")
	fp = strings.ReplaceAll(fp, `}`, "-")
	fp = strings.ReplaceAll(fp, `|`, "-")
	fp = strings.ReplaceAll(fp, `\`, "-")
	fp = strings.ReplaceAll(fp, `^`, "-")
	fp = strings.ReplaceAll(fp, `~`, "-")
	fp = strings.ReplaceAll(fp, `[`, "-")
	fp = strings.ReplaceAll(fp, `]`, "-")
	fp = strings.ReplaceAll(fp, `'`, "-")
	fp = strings.ReplaceAll(fp, `"`, "-")
	fp = strings.ReplaceAll(fp, "`", "-")
	fp = strings.ReplaceAll(fp, ",", "-")
	fp = strings.ReplaceAll(fp, ".", "-")

	fp = strings.ReplaceAll(fp, `----`, "-")
	fp = strings.ReplaceAll(fp, `---`, "-")
	fp = strings.ReplaceAll(fp, `--`, "-")
	fp = strings.ReplaceAll(fp, `--`, "-")
	fp = strings.ReplaceAll(fp, `--`, "-")
	fp = strings.TrimSuffix(fp, "---")
	fp = strings.TrimSuffix(fp, "--")
	fp = strings.TrimSuffix(fp, "-")

	var b [4]byte
	rand.Read(b[:])
	fp = fmt.Sprintf("/blog/posts/%s-z%x", fp, b)

	return fp
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
func processMarkdownFile(gc *GenerationContext, path string) (*types.Document, error) {
	log.Debug().Str("path", path).Msgf("start processing markdown file %s", path)

	log.Debug().Str("path", path).Msgf("start reading markdown file %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("path", path).Int("size", len(data)).Msgf("read markdown file %s", path)

	doc, err := parseMarkdown(path, data)
	if err != nil {
		return nil, err
	}

	if doc.Metadata.ID == "" {
		doc.Metadata.ID = types.RandID()
		log.Debug().Str("path", path).Str("id", doc.Metadata.ID).Msgf("assigned new ID to document %s", path)
	}

	if doc.Metadata.Date.IsZero() {
		doc.Metadata.Date = time.Now().UTC()
		log.Debug().Str("path", path).Msgf("assigned new date to document %s", path)
	}

	if doc.Metadata.Path == "" {
		doc.Metadata.Path = generatePath(doc.Metadata.Title)
	}

	if doc.Metadata.Description == "" {
		log.Debug().Str("path", path).Msgf("generating description for document %s", path)
		desc, err := description.GenerateDescription(context.Background(), llmModel, doc.Markdown)
		if err != nil {
			log.Error().Str("path", path).Msgf("failed to generate description for document %s", path)
		}
		doc.Metadata.Description = desc
		log.Debug().Str("path", path).Str("description", doc.Metadata.Description).Msgf("generated description for document %s", path)
	}

	if doc.Metadata.Language == "" {
		log.Debug().Str("path", path).Msgf("detecting language of document %s", path)
		detectedLang, ok := languageDetector.DetectLanguageOf(doc.Markdown)
		lang := "en"
		if ok {
			switch detectedLang {
			case lingua.English:
				lang = "en"
			case lingua.Spanish:
				lang = "es"
			case lingua.Chinese:
				lang = "zh"
			case lingua.Korean:
				lang = "ko"
			case lingua.Japanese:
				lang = "ja"
			case lingua.German:
				lang = "de"
			case lingua.Russian:
				lang = "ru"
			case lingua.French:
				lang = "fr"
			case lingua.Dutch:
				lang = "nl"
			case lingua.Italian:
				lang = "it"
			case lingua.Indonesian:
				lang = "id"
			case lingua.Portuguese:
				lang = "pt"
			case lingua.Swedish:
				lang = "sv"
			}
			confidence := languageDetector.ComputeLanguageConfidence(doc.Markdown, detectedLang)
			log.Debug().Str("path", path).Str("lang", lang).Float64("confidence", confidence).Msgf("detected language of document %s", path)
			doc.Metadata.Language = lang
		}
	}

	log.Debug().Str("path", path).Msgf("saving updated document %s", path)

	if doc.Type == types.DocumentTypeMarkdown {
		newMeta, err := yaml.Marshal(&doc.Metadata)
		if err != nil {
			return nil, err
		}

		original := doc.Markdown
		original = strings.TrimPrefix(original, "---\n")
		_, origDocument, ok := strings.Cut(original, "---\n")
		if !ok {
			return nil, ErrInvalidMarkdown
		}
		newDocument := "---\n" + string(newMeta) + "---\n" + origDocument
		doc.Markdown = newDocument

		fStat, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		err = os.WriteFile(path, []byte(doc.Markdown), fStat.Mode())
		if err != nil {
			return nil, err
		}
		log.Debug().Str("path", path).Msgf("saved updated document %s", path)
	} else {
		log.Debug().Str("path", path).Msgf("skipping non-markdown document %s", path)
	}

	now := time.Now()

	// Update Post Object
	var post *types.Post
	if p, ok := gc.DataStore.Posts[doc.Metadata.ID]; ok {
		post = p
	} else {
		post = &types.Post{
			ID:         doc.Metadata.ID,
			CreatedAt:  now,
			UpdatedAt:  now,
			Translated: make(map[string]*types.Document),
		}
		gc.DataStore.Posts[doc.Metadata.ID] = post
	}

	hash := doc.Hash()
	post.FilePath = path
	post.Path = doc.Metadata.Path
	post.Main = doc
	if post.Hash != hash {
		post.Hash = hash
		post.UpdatedAt = now
	}

	if gc.UsedPosts == nil {
		gc.UsedPosts = make(map[string]struct{})
	}
	gc.UsedPosts[post.ID] = struct{}{}

	if gc.PathMap == nil {
		gc.PathMap = make(map[string]string)
	}
	gc.PathMap[post.Path] = post.ID

	log.Debug().Str("path", path).Msgf("done processing markdown file %s", path)
	return doc, nil
}

func generatePostPages(gc *GenerationContext) error {
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
		log.Debug().Str("path", post.Path).Msgf("generating post page %s", post.Path)
		fp := filepath.Join(distDir, post.Path)
		err := os.MkdirAll(filepath.Dir(fp), 0755)
		if err != nil {
			return err
		}

		meta := &view.Metadata{
			Language:    post.Main.Metadata.Language,
			Title:       post.Main.Metadata.Title,
			Description: post.Main.Metadata.Description,
			Author:      post.Main.Metadata.Author,
			URL:         baseURL + post.Path,
			BaseURL:     baseURL,
			Canonical:   baseURL + post.Path,
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
			fp += "index.html"
		} else {
			fp += ".html"
		}

		err = os.WriteFile(fp, b.Bytes(), 0644)
		if err != nil {
			return err
		}
	}

	log.Debug().Msg("done generating post pages")
	return nil
}

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

	err = generatePostPages(gc)
	if err != nil {
		return err
	}

	err = minifyDir(distDir)
	if err != nil {
		return err
	}

	// Remove unused posts
	for id := range gc.DataStore.Posts {
		if _, ok := gc.UsedPosts[id]; !ok {
			log.Debug().Str("id", id).Msgf("removing unused post %s", id)
			delete(gc.DataStore.Posts, id)
		}
	}

	log.Debug().Msg("done generating website")
	return nil
}

func main() {
	defer llmClient.Close()
	defer llmModel.Close()

	_, err := os.Stat(dbFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal().Err(err).Msgf("failed to stat database file %s", dbFile)
	}

	var f *os.File
	if err != nil && os.IsNotExist(err) {
		log.Info().Err(err).Msgf("database file %s does not exist, Creating new database file", dbFile)
		f, err = os.OpenFile(dbFile, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to create database file %s", dbFile)
		}

		w, err := zstd.NewWriter(f)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to create zstd writer for database file %s", dbFile)
		}

		_, err = w.Write([]byte("{}"))
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to write to database file %s", dbFile)
		}

		err = w.Close()
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to close zstd writer for database file %s", dbFile)
		}

		_, err = f.Seek(0, 0)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to seek to beginning of database file %s", dbFile)
		}
	} else {
		f, err = os.OpenFile(dbFile, os.O_RDWR, 0644)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to open database file %s", dbFile)
		}
	}

	var gc GenerationContext
	var ds DataStore
	gc.DataStore = &ds

	r, err := zstd.NewReader(f)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to create zstd reader for database file %s", dbFile)
	}

	err = json.NewDecoder(r).Decode(&ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to decode database file %s", dbFile)
	}
	r.Close()

	err = f.Close()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to close database file %s", dbFile)
	}

	if gc.DataStore == nil {
		gc.DataStore = &DataStore{}
	}

	if gc.DataStore.Posts == nil {
		gc.DataStore.Posts = make(map[string]*types.Post)
	}

	err = generate(&gc)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to generate website")
	}

	// Update Database
	f, err = os.OpenFile(dbFile+".tmp", os.O_CREATE|os.O_RDWR|os.O_TRUNC|os.O_EXCL, 0644)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to create temporary database file %s", dbFile)
	}

	w, err := zstd.NewWriter(f, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to create zstd writer for database file %s", dbFile)
	}

	err = json.NewEncoder(w).Encode(&ds)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to encode database file %s", dbFile)
	}

	err = w.Close()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to close zstd writer for database file %s", dbFile)
	}

	err = f.Close()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to close database file %s", dbFile)
	}

	err = os.Rename(dbFile+".tmp", dbFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to rename temporary database file %s", dbFile)
	}

	log.Info().Msgf("database file %s updated", dbFile)
	log.Info().Msgf("website generated")

	// print database as JSON
	jsonData, err := json.MarshalIndent(&gc, "", "  ")
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to marshal database file %s", dbFile)
	}
	fmt.Println(string(jsonData))
}
