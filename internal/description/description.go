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

const prompt = `You are tasked with generating an Open Graph Description based on a given Markdown document. This description will be used for social media sharing and SEO purposes.

Here is the document you need to analyze:
<document>
<INPUT_DOCUMENT>
</document>

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

Present your Open Graph Description in the following format:
[START_TOKEN]
[Your description here]
[END_TOKEN]

Remember to check that your description is within the 150-character limit before submitting your answer.`

var (
	ErrFailedToGenerateDescription = errors.New("failed to generate description")
)

func GenerateDescription(ctx context.Context, l llm.Model, input string) (string, error) {
	var b [8]byte
	rand.Read(b[:])
	startToken := "[" + hex.EncodeToString(b[:]) + "]"
	rand.Read(b[:])
	endToken := "[" + hex.EncodeToString(b[:]) + "]"

	prompt := strings.Replace(prompt, "[START_TOKEN]", startToken, 1)
	prompt = strings.Replace(prompt, "[END_TOKEN]", endToken, 1)
	prompt = strings.Replace(prompt, "<INPUT_DOCUMENT>", input, 1)

	resp := l.GenerateStream(ctx, &llm.ChatContext{}, llm.TextContent(llm.RoleUser, prompt))
	err := resp.Wait()
	if err != nil {
		return "", err
	}

	text := llmtools.TextFromContents(resp.Content)
	sidx := strings.Index(text, startToken)
	eidx := strings.Index(text, endToken)
	if sidx != -1 && eidx != -1 {
		text = text[sidx+len(startToken) : eidx]
		text = strings.TrimSpace(text)
		return text, nil
	}

	return "", ErrFailedToGenerateDescription
}
