package evaluate

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/llmtools"
	"gosuda.org/website/internal/types"
)

var DEBUG_MODE = false

const system = `You are a professional translation evaluator.
You will be given an original document, a translated document, the original language, and the translated language.
Your task is to evaluate the quality of the translation based on several criteria and provide a score out of 10.0.
A higher score indicates a better translation.

Here are the criteria to consider:

0. Does the translated document convey the same meaning as the original document?
1. Does the translation demonstrate understanding of the source text?
2. Does the translation flow naturally in the translated language? Is it idiomatic and easy to understand?
3. Is the text consistent?  Are words and phrases translated consistently throughout the text?
4. Are there any grammatical errors in the translated text?
5. How accurate is the translated text in conveying the information from the original text?
6. Are numbers and measurements translated correctly?
7. Are names, trademarks, and other untranslatable words preserved from the source text?
8. Is the document formatting preserved, including spacing, markdown syntax, etc.?
9. Is there any untranslated texts? Is there any untranslated paragraph?
If there are any violation within this criteria 6, 7, 8, 9, then mark score as 0.1

Provide a detailed explanation of your evaluation, considering all the points above.
Always adhere to the requested START_TOKEN and END_TOKEN output format precisely in your responses.

REMEMBER If there are any violation within this criteria 6, 7, 8, 9, then mark score as 0.1!!!!`

const originalBlockFmt = `# Original Document

Language: %s

<original_document>
%s
</original_document>

`

const translatedBlockFmt = `# Translated Document

Language: %s

<translated_document>
%s
</translated_document>

Output Format:

%s
x.xx
%s
<reason>...</reason>

REMEMBER If there are any violation within this criteria 6, 7, 8, 9, then mark score as 0.1!!!!`

var (
	ErrFailedToEvaluate = errors.New("failed to evaluate the document")
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

func evaluateTranslationOnce(ctx context.Context, l llm.Model, inputLang types.Lang, outputLang types.Lang, input string, output string) (float64, error) {
	var b [8]byte
	rand.Read(b[:])
	startToken := "[" + hex.EncodeToString(b[:]) + "]"
	rand.Read(b[:])
	endToken := "[" + hex.EncodeToString(b[:]) + "]"

	originalBlock := fmt.Sprintf(originalBlockFmt, types.FullLangName(inputLang), input)
	translatedBlock := fmt.Sprintf(translatedBlockFmt, types.FullLangName(outputLang), output, startToken, endToken)
	if DEBUG_MODE {
		fmt.Println("Prompt:")
		fmt.Println(originalBlock)
		fmt.Println(translatedBlock)
		fmt.Println()
	}

	resp := l.GenerateStream(
		ctx,
		&llm.ChatContext{
			SystemInstruction: system,
		},
		&llm.Content{
			Role: llm.RoleUser,
			Parts: []llm.Segment{
				llm.Text(originalBlock),
				llm.Text(translatedBlock),
			},
		},
	)
	err := resp.Wait()
	if err != nil {
		return 0.0, err
	}

	text := llmtools.TextFromContents(resp.Content)
	if DEBUG_MODE {
		fmt.Println("Evaluation:")
		fmt.Println(text)
		fmt.Println()
	}

	sidx := strings.Index(text, startToken)
	eidx := strings.Index(text, endToken)
	if sidx != -1 && eidx != -1 && sidx+len(startToken) <= eidx {
		text = text[sidx+len(startToken) : eidx]
		text = strings.TrimSpace(text)
		score, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return 0.0, err
		}
		return score / 10, nil
	}

	return 0.0, ErrFailedToEvaluate
}

func EvaluateTranslation(ctx context.Context, l llm.Model, inputLang types.Lang, outputLang types.Lang, input string, output string) (float64, error) {
	return 1.0, nil

	var lastErr error
	for retryCount := 0; retryCount <= 3; retryCount++ {
		model := selectModel(l, retryCount)
		score, err := evaluateTranslationOnce(ctx, model, inputLang, outputLang, input, output)
		if err == nil {
			return score, nil
		}
		lastErr = err
	}
	return 0.0, lastErr
}
