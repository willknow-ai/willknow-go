# Provider Package

这个包提供了统一的AI模型提供商接口，支持多个AI服务商。

## 支持的提供商

### 1. Anthropic (Claude)

使用Anthropic的Claude模型。

```go
import "github.com/willknow-ai/willknow-go/provider"

// 创建Anthropic provider
p := provider.NewAnthropicProvider("your-api-key", "claude-sonnet-4-5-20250929")

// 或使用工厂方法
p, err := provider.NewProvider(provider.ProviderAnthropic, "your-api-key", "claude-sonnet-4-5-20250929")
```

默认模型：`claude-sonnet-4-5-20250929`

API文档：https://docs.anthropic.com/

### 2. DeepSeek

使用DeepSeek的模型（OpenAI兼容格式）。

```go
import "github.com/willknow-ai/willknow-go/provider"

// 创建DeepSeek provider
p := provider.NewDeepSeekProvider("your-api-key", "deepseek-chat")

// 或使用工厂方法
p, err := provider.NewProvider(provider.ProviderDeepSeek, "your-api-key", "deepseek-chat")
```

默认模型：`deepseek-chat`

API文档：https://platform.deepseek.com/

## 接口定义

```go
type Provider interface {
    // 发送消息并返回完整响应
    SendMessage(messages []Message, tools []Tool, system string) (*Response, error)
    
    // 发送消息并返回流式响应
    SendMessageStream(messages []Message, tools []Tool, system string) (io.ReadCloser, error)
    
    // 获取提供商名称
    GetName() string
}
```

## 使用示例

```go
package main

import (
    "fmt"
    "github.com/willknow-ai/willknow-go/provider"
)

func main() {
    // 创建provider
    p, err := provider.NewProvider(
        provider.ProviderAnthropic,
        "your-api-key",
        "", // 留空使用默认模型
    )
    if err != nil {
        panic(err)
    }

    // 准备消息
    messages := []provider.Message{
        {
            Role: "user",
            Content: []provider.ContentBlock{
                {
                    Type: "text",
                    Text: "Hello, how are you?",
                },
            },
        },
    }

    // 发送消息
    response, err := p.SendMessage(messages, nil, "You are a helpful assistant")
    if err != nil {
        panic(err)
    }

    // 处理响应
    for _, block := range response.Content {
        if block.Type == "text" {
            fmt.Println(block.Text)
        }
    }
}
```

## 添加新的提供商

1. 在`provider`包中创建新文件，例如`openai.go`
2. 实现`Provider`接口
3. 在`factory.go`中添加新的provider类型
4. 更新文档

示例结构：

```go
package provider

type OpenAIProvider struct {
    apiKey     string
    model      string
    httpClient *http.Client
}

func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
    if model == "" {
        model = "gpt-4"
    }
    return &OpenAIProvider{
        apiKey:     apiKey,
        model:      model,
        httpClient: &http.Client{},
    }
}

func (p *OpenAIProvider) GetName() string {
    return "OpenAI"
}

func (p *OpenAIProvider) SendMessage(messages []Message, tools []Tool, system string) (*Response, error) {
    // 实现逻辑
}

func (p *OpenAIProvider) SendMessageStream(messages []Message, tools []Tool, system string) (io.ReadCloser, error) {
    // 实现逻辑
}
```

## 注意事项

1. **API格式差异**：不同提供商的API格式可能不同，需要进行格式转换
2. **工具调用**：某些提供商可能不支持工具调用（function calling）
3. **流式响应**：流式响应的格式在不同提供商之间可能有差异
4. **错误处理**：确保正确处理各种API错误情况
