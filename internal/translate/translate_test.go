package translate

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/lemon-mint/coord/llm"
)

func TestChunkMarkdown(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name: "Original test case",
			input: `# Test Markdown

This is a paragraph with some text.

## Code Block Test

` + "```go" + `
func example() {
	fmt.Println("Hello, World!")
}
` + "```" + `

Another paragraph after the code block.

- List item 1
- List item 2

> This is a blockquote.

Final paragraph with a [link](https://example.com).`,
		},
		{
			name: "Long Original test case",
			input: strings.Repeat(`# Test Markdown

This is a paragraph with some text.

## Code Block Test

`+"```go"+`
func example() {
	fmt.Println("Hello, World!")
}
`+"```"+`

Another paragraph after the code block.

- List item 1
- List item 2

> This is a blockquote.

Final paragraph with a [link](https://example.com).`, 16000),
		},
		{
			name: "Long paragraphs and multiple code blocks",
			input: `# Long Content Test

This is a very long paragraph that should exceed the token limit. It contains a lot of text to ensure that the chunkMarkdown function correctly handles long content. We want to make sure that it splits the content appropriately without breaking the markdown structure. This paragraph goes on and on with more text to really push the limits of our function.

## First Code Block

` + "```python" + `
def long_function():
    print("This is a long function")
    for i in range(100):
        print(f"Iteration {i}")
    return "Done"
` + "```" + `

Another paragraph here to separate the code blocks. We'll add more text to make it longer and ensure proper chunking behavior.

## Second Code Block

` + "```javascript" + `
function anotherLongFunction() {
    console.log("This is another long function");
    for (let i = 0; i < 100; i++) {
        console.log('Iteration ${i}');
    }
    return "Done";
}
` + "```" + `

Final paragraph after multiple code blocks to test handling of mixed content types.`,
		},
		{
			name: "Nested lists and block quotes",
			input: `# Nested Structures Test

1. First level item
   - Nested unordered item
   - Another nested item
     1. Deep nested ordered item
     2. Another deep nested item
2. Second level item
   > This is a block quote inside a list
   > It continues for multiple lines
   > To test proper handling of nested structures

- Unordered list item
  > Another block quote
  > With multiple lines

  And some text after the quote

3. Third level item
   ` + "```" + `
   This is a code block inside a list item
   It should be handled correctly
   ` + "```" + `

Final paragraph to conclude the nested structures test.`,
		},
		{
			name: "Headers, horizontal rules, and tables",
			input: `# Main Header

## Subheader 1

Some content under subheader 1.

### Sub-subheader

More content here.

---

## Subheader 2

Content under subheader 2.

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Row 1, Col 1 | Row 1, Col 2 | Row 1, Col 3 |
| Row 2, Col 1 | Row 2, Col 2 | Row 2, Col 3 |
| Row 3, Col 1 | Row 3, Col 2 | Row 3, Col 3 |

#### Deep nested header

* List item 1
* List item 2

---

# Another Main Header

Final paragraph to conclude the headers and tables test.`,
		},
		{
			name: "Very long content",
			input: strings.Repeat("This is a long sentence that will be repeated many times to create a very large content. ", 2000) +
				"\n\n" + strings.Repeat("Another long sentence with different words to increase variety in the content. ", 2000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("Running test case: %s\n", tc.name)
			chunks := chunkMarkdown(tc.input)
			fmt.Printf("Number of chunks: %d\n", len(chunks))

			joined := strings.Join(chunks, "")

			if joined != tc.input {
				t.Errorf("Joined chunks do not match original input for test case '%s'.\nExpected length: %d\nGot length: %d", tc.name, len(tc.input), len(joined))

				// Find the first difference
				for i := 0; i < len(tc.input) && i < len(joined); i++ {
					if tc.input[i] != joined[i] {
						fmt.Printf("First difference at index %d. Expected: %q, Got: %q\n", i, tc.input[i], joined[i])
						break
					}
				}
			} else {
				fmt.Printf("Test case '%s' passed successfully\n", tc.name)
			}

			// Additional check for very long content
			if tc.name == "Very long content" {
				if len(chunks) < 2 {
					t.Errorf("Expected multiple chunks for very long content, got %d chunks", len(chunks))
				} else {
					fmt.Printf("Very long content split into %d chunks as expected\n", len(chunks))
				}
			}

			fmt.Println() // Add a blank line between test cases
		})
	}
}

func TestBetween(t *testing.T) {
	cases := []struct {
		text, want string
		ok         bool
	}{
		{"x[s]hello[e]y", "hello", true},
		{"[e] echoed first [s]hello[e]", "hello", true},
		{"[e] only end", "", false},
		{"[s] no end", "", false},
	}
	for _, c := range cases {
		got, ok := between(c.text, "[s]", "[e]")
		if got != c.want || ok != c.ok {
			t.Errorf("between(%q) = %q, %v; want %q, %v", c.text, got, ok, c.want, c.ok)
		}
	}
}

type mockModel struct {
	name string
	fn   func(chat *llm.ChatContext, input *llm.Content) (*llm.StreamContent, error)
}

func (m *mockModel) Name() string  { return m.name }
func (m *mockModel) Close() error { return nil }
func (m *mockModel) GenerateStream(_ context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	ch := make(chan llm.Segment)
	close(ch)
	resp, err := m.fn(chat, input)
	if err != nil {
		return &llm.StreamContent{
			Err:     err,
			Content: &llm.Content{},
			Stream:  ch,
		}
	}
	resp.Stream = ch
	return resp
}

type mockCascade struct {
	models []llm.Model
}

func (c *mockCascade) ModelForAttempt(attempt int) llm.Model {
	return c.models[attempt%len(c.models)]
}
func (c *mockCascade) Name() string  { return "cascade" }
func (c *mockCascade) Close() error { return nil }
func (c *mockCascade) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	return c.models[0].GenerateStream(ctx, chat, input)
}

func TestCascadeRetryOrder(t *testing.T) {
	var called []string
	m1 := &mockModel{
		name: "anthropic/claude-haiku-5.5",
		fn: func(_ *llm.ChatContext, _ *llm.Content) (*llm.StreamContent, error) {
			called = append(called, "anthropic/claude-haiku-5.5")
			return nil, errors.New("simulated failure")
		},
	}
	m2 := &mockModel{
		name: "google/gemini-3.5-flash-lite",
		fn: func(_ *llm.ChatContext, _ *llm.Content) (*llm.StreamContent, error) {
			called = append(called, "google/gemini-3.5-flash-lite")
			return nil, errors.New("simulated failure")
		},
	}

	cascade := &mockCascade{models: []llm.Model{m1, m2}}
	_, err := Translate(context.Background(), cascade, "Hello world", "Korean")
	if err == nil {
		t.Fatal("expected error when all attempts fail")
	}

	want := []string{
		"anthropic/claude-haiku-5.5",
		"google/gemini-3.5-flash-lite",
		"anthropic/claude-haiku-5.5",
		"google/gemini-3.5-flash-lite",
	}
	if !reflect.DeepEqual(called, want) {
		t.Fatalf("called models = %v, want %v", called, want)
	}
}

func TestSplitSurroundingWhitespace(t *testing.T) {
	cases := []struct {
		in      string
		leading string
		core    string
		trail   string
	}{
		{"\n## Title\n\nParagraph\n\n", "\n", "## Title\n\nParagraph", "\n\n"},
		{"   ```bash\n   echo hi\n   ```\n\n", "   ", "```bash\n   echo hi\n   ```", "\n\n"},
		{"No whitespace", "", "No whitespace", ""},
		{"\n\t  \n", "\n\t  \n", "", ""},
	}
	for _, tc := range cases {
		lead, core, trail := splitSurroundingWhitespace(tc.in)
		if lead != tc.leading || core != tc.core || trail != tc.trail {
			t.Errorf("splitSurroundingWhitespace(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tc.in, lead, core, trail, tc.leading, tc.core, tc.trail)
		}
		if lead+core+trail != tc.in {
			t.Errorf("reconstructed %q != %q", lead+core+trail, tc.in)
		}
	}
}

func TestTranslatePreservesChunkWhitespace(t *testing.T) {
	// Simulate Haiku stripping trailing newlines or injecting extra newlines inside magic tags
	m := &mockModel{
		name: "anthropic/claude-haiku-5.5",
		fn: func(chat *llm.ChatContext, input *llm.Content) (*llm.StreamContent, error) {
			if chat == nil || chat.SystemInstruction == "" {
				t.Errorf("expected non-empty SystemInstruction for prompt caching")
			}

			if len(input.Parts) != 2 {
				t.Fatalf("expected 2 content parts, got %d", len(input.Parts))
			}
			sourceBlock := string(input.Parts[0].(llm.Text))
			payload := strings.TrimPrefix(sourceBlock, "INPUT_TEXT:\n")
			payload = strings.TrimSuffix(payload, "\n\n")
			startTag := payload[:18]
			endTag := payload[len(payload)-18:]
			inner := payload[18 : len(payload)-18]

			if strings.HasPrefix(inner, "\n") || strings.HasSuffix(inner, "\n") {
				t.Errorf("expected stripped inner payload without surrounding newlines, got %q", inner)
			}

			// Model injects arbitrary leading/trailing newlines inside tags
			out := startTag + "\n\n" + inner + " (translated)\n" + endTag
			return &llm.StreamContent{
				Content: llm.TextContent(llm.RoleModel, out),
			}, nil
		},
	}

	chunks := []string{
		"\n## Heading\n\nFirst paragraph.\n\n",
		"```go\nfunc main() {}\n```\n\n",
		"   - Indented item\n\n",
		"\n" + strings.Repeat("Long cacheable paragraph content. ", 80) + "\n\n",
	}
	var translated []string
	for _, ch := range chunks {
		out, err := translateChunk(context.Background(), m, ch, "Korean")
		if err != nil {
			t.Fatalf("translateChunk(%q) failed: %v", ch, err)
		}
		translated = append(translated, out)
	}
	got := strings.Join(translated, "")
	want := "\n## Heading\n\nFirst paragraph. (translated)\n\n" +
		"```go\nfunc main() {}\n``` (translated)\n\n" +
		"   - Indented item (translated)\n\n" +
		"\n" + strings.TrimSpace(strings.Repeat("Long cacheable paragraph content. ", 80)) + " (translated) \n\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}
