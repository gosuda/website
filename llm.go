package main

import (
	"context"
	"os"
	"time"

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
	"golang.org/x/time/rate"
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
		lingua.Czech,
	}

	languageDetector = lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		Build()

	if os.Getenv("LLM_INIT") == "false" || os.Getenv("LLM_INIT") == "0" {
		log.Info().Msg("llm init skipped")
		return
	}

	var err error
	var client provider.LLMClient

	if os.Getenv("PROVIDER") == "aistudio" {
		log.Debug().Msg("initializing llm client")
		client, err = coord.NewLLMClient(
			context.Background(),
			"aistudio",
			pconf.WithAPIKey(os.Getenv("AI_STUDIO_API_KEY")),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create llm client")
		}
	} else {
		log.Debug().Str("location", os.Getenv("LOCATION")).Str("project_id", os.Getenv("PROJECT_ID")).Msg("initializing llm client")
		client, err = coord.NewLLMClient(
			context.Background(),
			"vertexai",
			pconf.WithLocation(os.Getenv("LOCATION")),
			pconf.WithProjectID(os.Getenv("PROJECT_ID")),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create llm client")
		}
	}
	llmClient = client
	log.Debug().Msg("llm client initialized")

	llmModel, err = llmClient.NewLLM("gemini-2.5-flash", &llm.Config{
		Temperature:           Ptr(float32(0.7)),
		MaxOutputTokens:       Ptr(65535),
		SafetyFilterThreshold: llm.BlockOff,
		ThinkingConfig: &llm.ThinkingConfig{
			ThinkingBudget: pconf.Ptrify(0),
		},
	})

	if err != nil {
		log.Fatal().Err(err).Msg("failed to create llm model")
	}

	//llmModel = newRateLimitModel(llmModel, rate.Every(time.Minute/9))
	log.Debug().Msg("llm model initialized")
}

func Ptr[T any](t T) *T {
	return &t
}

type rateLimitModel struct {
	llm.Model
	limit   rate.Limit
	limiter *rate.Limiter
}

func newRateLimitModel(model llm.Model, limit rate.Limit) *rateLimitModel {
	return &rateLimitModel{
		Model:   model,
		limit:   limit,
		limiter: rate.NewLimiter(limit, 1),
	}
}

func (r *rateLimitModel) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	if err := r.limiter.Wait(ctx); err != nil {
		ch := make(chan llm.Segment)
		close(ch)
		return &llm.StreamContent{
			Err:          err,
			Content:      &llm.Content{},
			FinishReason: llm.FinishReasonError,
			Stream:       ch,
		}
	}
	return r.Model.GenerateStream(ctx, chat, input)
}
