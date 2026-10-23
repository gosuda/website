package description

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/llmtools"
)

const systemPrompt = `You are tasked with generating an Open Graph Description based on a given Markdown document. This description will be used for social media sharing and SEO purposes.

An Open Graph Description is a brief summary of a web page's content, typically displayed when the page is shared on social media platforms. It should accurately represent the core content of the document while being engaging and informative.

To complete this task:
1. Carefully read and analyze the provided Markdown document.
2. Identify the main topic, key points, and overall message of the content.
3. Consider the document's structure, headings, and any emphasized text to determine the most important information.

When writing the Open Graph Description:
- Create a single, concise sentence that summarizes the core content of the document.
- Keep the description within 150 characters.
- Write in a clear and engaging style that encourages users to click and read more.
- Optimize for SEO by including relevant keywords naturally, but avoid keyword stuffing.
- Ensure the description accurately represents the document's content.
- Write the description in the same language as the input document.
- Never use the em-dash ("—") or spaced en-dash (" – ").

Remember to check that your description is within the 150-character limit before submitting your answer.`

var (
	ErrFailedToGenerateDescription = errors.New("failed to generate description")
)

type attemptModelSelector interface {
	ModelForAttempt(attempt int) llm.Model
}

func selectModel(l llm.Model, attempt int) llm.Model {
	if s, ok := l.(attemptModelSelector); ok {
		if m := s.ModelForAttempt(attempt); m != nil {
			return m
		}
	}
	return l
}

func generateDescriptionOnce(ctx context.Context, l llm.Model, input string) (string, error) {
	var b [8]byte
	rand.Read(b[:])
	startToken := "[" + hex.EncodeToString(b[:]) + "]"
	rand.Read(b[:])
	endToken := "[" + hex.EncodeToString(b[:]) + "]"

	docBlock := "<document>\n" + input + "\n</document>\n\n"
	instructionBlock := "Present your Open Graph Description in the following format:\n" +
		startToken + "\n[Your description here]\n" + endToken

	resp := l.GenerateStream(
		ctx,
		&llm.ChatContext{
			SystemInstruction: systemPrompt,
		},
		&llm.Content{
			Role: llm.RoleUser,
			Parts: []llm.Segment{
				llm.Text(docBlock),
				llm.Text(instructionBlock),
			},
		},
	)
	err := resp.Wait()
	if err != nil {
		return "", err
	}

	text := llmtools.TextFromContents(resp.Content)
	sidx := strings.Index(text, startToken)
	eidx := strings.Index(text, endToken)
	if sidx != -1 && eidx != -1 && sidx+len(startToken) <= eidx {
		text = text[sidx+len(startToken) : eidx]
		text = strings.TrimSpace(text)
		return text, nil
	}

	return "", ErrFailedToGenerateDescription
}

func GenerateDescription(ctx context.Context, l llm.Model, input string) (string, error) {
	var lastErr error
	for retryCount := 0; retryCount <= 3; retryCount++ {
		model := selectModel(l, retryCount)
		desc, err := generateDescriptionOnce(ctx, model, input)
		if err == nil {
			return desc, nil
		}
		lastErr = err
	}
	return "", lastErr
}
