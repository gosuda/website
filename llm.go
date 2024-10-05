package main

import (
	"context"
	"os"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	_ "github.com/lemon-mint/coord/provider/aistudio"
	_ "github.com/lemon-mint/coord/provider/anthropic"
	_ "github.com/lemon-mint/coord/provider/openai"
	_ "github.com/lemon-mint/coord/provider/vertexai"
	"github.com/pemistahl/lingua-go"
	"github.com/rs/zerolog/log"
)

var llmClient provider.LLMClient
var llmModel llm.Model
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

func Ptr[T any](t T) *T {
	return &t
}
