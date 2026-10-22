package main

import (
	"context"
	"errors"
	"os"
	"strings"

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
	"gopkg.eu.org/envloader"
)

const (
	openRouterBaseURL       = "https://openrouter.ai/api/v1"
	openRouterPrimaryModel  = "anthropic/claude-haiku-5.5"
	openRouterFallbackModel = "google/gemini-3.5-flash-lite"
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

	envloader.LoadEnvFile(".env")

	if os.Getenv("LLM_INIT") == "false" || os.Getenv("LLM_INIT") == "0" {
		log.Info().Msg("llm init skipped")
		return
	}

	var err error
	var client provider.LLMClient

	providerName := os.Getenv("PROVIDER")
	switch providerName {
	case "aistudio":
		log.Debug().Msg("initializing llm client (aistudio)")
		client, err = coord.NewLLMClient(
			context.Background(),
			"aistudio",
			pconf.WithAPIKey(os.Getenv("AI_STUDIO_API_KEY")),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create llm client")
		}
	case "vertexai":
		log.Debug().Str("location", os.Getenv("LOCATION")).Str("project_id", os.Getenv("PROJECT_ID")).Msg("initializing llm client (vertexai)")
		client, err = coord.NewLLMClient(
			context.Background(),
			"vertexai",
			pconf.WithLocation(os.Getenv("LOCATION")),
			pconf.WithProjectID(os.Getenv("PROJECT_ID")),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create llm client")
		}
	default:
		log.Debug().Msg("initializing llm client (openrouter)")
		client, err = coord.NewLLMClient(
			context.Background(),
			"openai",
			pconf.WithBaseURL(openRouterBaseURL),
			pconf.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create llm client")
		}
	}
	llmClient = client
	log.Debug().Msg("llm client initialized")

	if providerName == "aistudio" || providerName == "vertexai" {
		llmModel, err = llmClient.NewLLM("gemini-3.5-flash-lite", &llm.Config{
			Temperature:           Ptr(float32(0.7)),
			MaxOutputTokens:       Ptr(65535),
			SafetyFilterThreshold: llm.BlockOff,
			ThinkingConfig: &llm.ThinkingConfig{
				ThinkingLevel: Ptr(llm.ThinkingLevelMinimal),
			},
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create llm model")
		}
	} else {
		primaryModel, err := llmClient.NewLLM(openRouterPrimaryModel, &llm.Config{
			Temperature:     Ptr(float32(0.7)),
			MaxOutputTokens: Ptr(65535),
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create primary llm model")
		}

		fallbackModel, err := llmClient.NewLLM(openRouterFallbackModel, &llm.Config{
			Temperature:     Ptr(float32(0.7)),
			MaxOutputTokens: Ptr(65535),
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create fallback llm model")
		}

		llmModel = newCascadeModel(
			&openRouterModel{Model: primaryModel},
			&openRouterModel{Model: fallbackModel},
		)
	}

	//llmModel = newRateLimitModel(llmModel, rate.Every(time.Minute/9))
	log.Debug().Str("model", llmModel.Name()).Msg("llm model initialized")
}

func Ptr[T any](t T) *T {
	return &t
}

type openRouterModel struct {
	llm.Model
}

func (m *openRouterModel) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	inner := m.Model.GenerateStream(ctx, chat, input)
	ch := make(chan llm.Segment, 128)
	out := &llm.StreamContent{
		Content: &llm.Content{},
		Stream:  ch,
	}
	go func() {
		defer close(ch)
		for seg := range inner.Stream {
			ch <- seg
		}
		out.Content = inner.Content
		out.UsageData = inner.UsageData
		out.FinishReason = inner.FinishReason
		out.Err = inner.Err
		if errors.Is(out.Err, llm.ErrNoResponse) && out.Content != nil && len(out.Content.Parts) > 0 {
			out.Err = nil
		}
	}()
	return out
}

type cascadeModel struct {
	models []llm.Model
}

func newCascadeModel(models ...llm.Model) *cascadeModel {
	return &cascadeModel{models: models}
}

func (c *cascadeModel) ModelForAttempt(attempt int) llm.Model {
	if len(c.models) == 0 {
		return nil
	}
	if attempt < 0 {
		attempt = 0
	}
	return c.models[attempt%len(c.models)]
}

func (c *cascadeModel) Name() string {
	names := make([]string, len(c.models))
	for i, m := range c.models {
		names[i] = m.Name()
	}
	return strings.Join(names, ",")
}

func (c *cascadeModel) Close() error {
	var firstErr error
	for _, m := range c.models {
		if err := m.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (c *cascadeModel) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	const maxAttempts = 4
	var lastResp *llm.StreamContent
	for attempt := 0; attempt < maxAttempts; attempt++ {
		m := c.ModelForAttempt(attempt)
		if attempt > 0 {
			log.Debug().Int("attempt", attempt+1).Str("model", m.Name()).Msg("retrying llm stream with cascade model")
		}
		resp := m.GenerateStream(ctx, chat, input)
		if err := resp.Wait(); err != nil {
			log.Warn().Err(err).Int("attempt", attempt+1).Str("model", m.Name()).Msg("llm stream attempt failed")
			lastResp = resp
			continue
		}
		return resp
	}
	return lastResp
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
