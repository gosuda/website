package translate

import (
	"fmt"
	"strings"
	"testing"
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
