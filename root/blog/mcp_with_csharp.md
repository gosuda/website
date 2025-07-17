---
id: a1fad13de24c90924cfa9e25e9f986cb
author: snowmerak
title: MCP host 조금 이해해보기
description: Anthropic의 MCP 프로토콜과 Go로 구현된 MCP 호스트(mcphost)에 대한 이해를 돕는 글입니다.
language: ko
date: 2025-04-08T00:59:42.416582136Z
path: /blog/posts/a-little-understanding-of-mcp-host-zb952aae0
---

## MCP가 무엇입니까

[MCP](https://modelcontextprotocol.io/introduction)란 Anthropic에서 claude를 위해 개발된 프로토콜입니다. MCP는 Model Context Protocol의 줄임말로써 LLM이 능동적으로 외부에 동작이나 리소스를 요청할 수 있도록 해주는 프로토콜입니다. MCP는 진짜 문자 그대로 요청과 응답을 주는 프로토콜에 불과하기 때문에 그 과정과 실행은 개발자가 해줘야 합니다.

### 내부 동작에 대해서

내부 동작에 대해 설명하기 앞서, [Gemini Function Calling](https://ai.google.dev/gemini-api/docs/function-calling)에 대해 짚고 넘어 가겠습니다. Gemini Function Calling도 MCP와 동일하게 LLM이 주도적으로 외부 동작을 호출할 수 있도록 합니다. 그럼 왜 Function Calling을 굳이 가져왔는가 의문이 들 것입니다. 굳이 가져온 이유는 Function Calling이 MCP보다 먼저 나오기도 했고, 동일하게 OpenAPI 스키마를 이용한다는 점에서 호환이 되어, 상호 간의 동작이 유사할 것으로 추측했습니다. 그렇다보니 비교적 Gemini Function Calling의 설명이 더욱 상세하기에 도움이 될 것으로 보여 가져왔습니다.

![FunctionCalling](/assets/images/mcp_with_csharp/function-calling-overview.png)

전체적인 흐름은 이렇습니다.

1. 함수를 정의합니다.
2. 프롬프트와 함께 Gemini에 함수 정의를 전송합니다.
   1. "Send user prompt along with the function declaration(s) to the model. It analyzes the request and determines if a function call would be helpful. If so, it responds with a structured JSON object."
3. Gemini가 필요하면 함수 호출을 요청합니다.
   1. Gemini가 필요하면 함수 호출을 위한 이름과 패러미터를 호출자가 전달받습니다.
   2. 호출자는 실행을 할지, 말지 정할 수 있습니다.
      1. 호출해서 정당한 값을 돌려줄 것인지
      2. 호출하지 않고 호출한 것처럼 데이터를 반환할지
      3. 그냥 무시할지
4. Gemini는 위 과정에서 한번에 여러개의 함수를 호출하거나, 함수 호출 후 결과를 보고 또 호출하는 등의 동작을 수행 및 요청합니다.
5. 결과적으로 정돈된 대답이 나오면 종료됩니다.

이 흐름은 일반적으로 MCP와 일맥상통합니다. 이는 [MCP의 튜토리얼](https://modelcontextprotocol.io/tutorials/building-mcp-with-llms)에서도 비슷하게 설명하고 있습니다. 이는 ollama tools도 비슷합니다.

그리고 정말 다행이게도 이 3가지 도구, ollama tools, MCP, Gemini Function Calling은 스키마 구조가 공유되다시피 해서 MCP 하나만 구현함으로 3곳에 다 쓸 수도 있다는 것입니다.

아 그리고 모두가 공유하는 단점이 있습니다. 결국 모델이 실행시켜주는 것이기 때문에 여러분이 쓰는 모델이 상태가 안 좋다면, 함수를 호출하지 않거나, 이상하게 호출한다거나, MCP 서버에 DOS를 날리는 등의 오동작을 할 수 있습니다.

## Go로 된 MCP 호스트

### mark3lab's mcphost

Go에는 mark3lab이란 조직에서 개발 중인 [mcphost](https://github.com/mark3labs/mcphost)가 있습니다.

사용법은 매우 간단합니다.

```sh
go install github.com/mark3labs/mcphost@latest
```

설치 후, `$HOME/.mcp.json` 파일을 만들어서 다음과 같이 작성합니다.

```json
{
  "mcpServers": {
    "sqlite": {
      "command": "uvx",
      "args": [
        "mcp-server-sqlite",
        "--db-path",
        "/tmp/foo.db"
      ]
    },
    "filesystem": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/server-filesystem",
        "/tmp"
      ]
    }
  }
}
```

그리고 다음과 같이 ollama 모델로 실행합니다.  
물론 그 전에 필요하면 `ollama pull mistral-small`로 모델을 받습니다.

> 기본적으로 claude나 qwen2.5를 추천하지만, 저는 현재로썬 mistral-small을 추천합니다.

```sh
mcphost -m ollama:mistral-small
```

다만 이렇게 실행하면, CLI 환경에서 질의응답 식으로만 사용할 수 있습니다.  
그렇기에 저희는 이 `mcphost`의 코드를 수정해서 좀 더 프로그래머블 하게 동작할 수 있게 수정해보겠습니다.

### mcphost 포크

이미 확인했다시피 `mcphost`에는 MCP를 활용해서 메타데이터를 추출하고, 함수를 호출하는 기능이 포함되어 있습니다. 그러므로 llm을 호출하는 부분, mcp 서버를 다루는 부분, 메시지 히스토리를 관리하는 부분이 필요합니다.

해당하는 부분을 가져온 것이 다음 패키지의 `Runner`입니다.

```go
package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/mark3labs/mcphost/pkg/history"
	"github.com/mark3labs/mcphost/pkg/llm"
)

type Runner struct {
	provider   llm.Provider
	mcpClients map[string]*mcpclient.StdioMCPClient
	tools      []llm.Tool

	messages []history.HistoryMessage
}
```

해당하는 부분의 내부 선언은 따로 보지 않겠습니다. 다만 거의 이름 그대로입니다.

```go
func NewRunner(systemPrompt string, provider llm.Provider, mcpClients map[string]*mcpclient.StdioMCPClient, tools []llm.Tool) *Runner {
	return &Runner{
		provider:   provider,
		mcpClients: mcpClients,
		tools:      tools,
		messages: []history.HistoryMessage{
			{
				Role: "system",
				Content: []history.ContentBlock{{
					Type: "text",
					Text: systemPrompt,
				}},
			},
		},
	}
}
```

여기에 쓰일 `mcpClients`와 `tools`에 대해서는 [해당 파일](https://github.com/snowmerak/mcphost/blob/main/internal/runner/from.go)을 확인해 주세요.  
`provider`는 ollama의 것을 쓸 테니 [해당 파일](https://github.com/snowmerak/mcphost/blob/main/pkg/llm/ollama/provider.go)을 확인해 주세요.

메인 요리는 `Run` 메서드입니다.

```go
func (r *Runner) Run(ctx context.Context, prompt string) (string, error) {
	if len(prompt) != 0 {
		r.messages = append(r.messages, history.HistoryMessage{
			Role: "user",
			Content: []history.ContentBlock{{
				Type: "text",
				Text: prompt,
			}},
		})
	}

	llmMessages := make([]llm.Message, len(r.messages))
	for i := range r.messages {
		llmMessages[i] = &r.messages[i]
	}

	const initialBackoff = 1 * time.Second
	const maxRetries int = 5
	const maxBackoff = 30 * time.Second

	var message llm.Message
	var err error
	backoff := initialBackoff
	retries := 0
	for {
		message, err = r.provider.CreateMessage(
			context.Background(),
			prompt,
			llmMessages,
			r.tools,
		)
		if err != nil {
			if strings.Contains(err.Error(), "overloaded_error") {
				if retries >= maxRetries {
					return "", fmt.Errorf(
						"claude is currently overloaded. please wait a few minutes and try again",
					)
				}

				time.Sleep(backoff)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				retries++
				continue
			}

			return "", err
		}

		break
	}

	var messageContent []history.ContentBlock

	var toolResults []history.ContentBlock
	messageContent = []history.ContentBlock{}

	if message.GetContent() != "" {
		messageContent = append(messageContent, history.ContentBlock{
			Type: "text",
			Text: message.GetContent(),
		})
	}

	for _, toolCall := range message.GetToolCalls() {
		input, _ := json.Marshal(toolCall.GetArguments())
		messageContent = append(messageContent, history.ContentBlock{
			Type:  "tool_use",
			ID:    toolCall.GetID(),
			Name:  toolCall.GetName(),
			Input: input,
		})

		parts := strings.Split(toolCall.GetName(), "__")

		serverName, toolName := parts[0], parts[1]
		mcpClient, ok := r.mcpClients[serverName]
		if !ok {
			continue
		}

		var toolArgs map[string]interface{}
		if err := json.Unmarshal(input, &toolArgs); err != nil {
			continue
		}

		var toolResultPtr *mcp.CallToolResult
		req := mcp.CallToolRequest{}
		req.Params.Name = toolName
		req.Params.Arguments = toolArgs
		toolResultPtr, err = mcpClient.CallTool(
			context.Background(),
			req,
		)

		if err != nil {
			errMsg := fmt.Sprintf(
				"Error calling tool %s: %v",
				toolName,
				err,
			)
			log.Printf("Error calling tool %s: %v", toolName, err)

			toolResults = append(toolResults, history.ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolCall.GetID(),
				Content: []history.ContentBlock{{
					Type: "text",
					Text: errMsg,
				}},
			})

			continue
		}

		toolResult := *toolResultPtr

		if toolResult.Content != nil {
			resultBlock := history.ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolCall.GetID(),
				Content:   toolResult.Content,
			}

			var resultText string
			for _, item := range toolResult.Content {
				if contentMap, ok := item.(map[string]interface{}); ok {
					if text, ok := contentMap["text"]; ok {
						resultText += fmt.Sprintf("%v ", text)
					}
				}
			}

			resultBlock.Text = strings.TrimSpace(resultText)

			toolResults = append(toolResults, resultBlock)
		}
	}

	r.messages = append(r.messages, history.HistoryMessage{
		Role:    message.GetRole(),
		Content: messageContent,
	})

	if len(toolResults) > 0 {
		r.messages = append(r.messages, history.HistoryMessage{
			Role:    "user",
			Content: toolResults,
		})

		return r.Run(ctx, "")
	}

	return message.GetContent(), nil
}
```

코드 자체는 [해당 파일](https://github.com/snowmerak/mcphost/blob/main/cmd/mcp.go)의 일부 코드를 짜집기 하였습니다.

내용은 대략 다음과 같습니다.

1. 프롬프트와 함께 툴 목록을 전송하여 실행 여부, 혹은 응답 생성을 요청합니다.
2. 응답이 생성되면 재귀를 멈추고 반환합니다.
3. LLM이 툴 실행 요청을 남긴다면, 호스트에서는 MCP Server를 호출합니다.
4. 응답을 히스토리에 추가해서 다시 1번으로 돌아갑니다.

## 끝으로

> 벌써 끝?

사실 할 말이 그렇게 많진 않습니다. 대략적으로 MCP Server가 어떻게 동작되는 지에 대한 이해를 도와 드리기 위해 작성된 글입니다. 이 글이 여러분들에게 자그맣게나마 MCP host의 동작을 이해함에 도움이 되었길 바랍니다.
