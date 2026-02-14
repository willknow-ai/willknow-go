package provider

// ProviderPreset defines configuration for a specific AI provider
type ProviderPreset struct {
	Name         string
	BaseURL      string
	DefaultModel string
}

// Presets contains predefined configurations for popular AI providers
var Presets = map[ProviderType]ProviderPreset{
	// Anthropic Claude
	ProviderAnthropic: {
		Name:         "Anthropic",
		BaseURL:      "https://api.anthropic.com/v1/messages",
		DefaultModel: "claude-sonnet-4-5-20250929",
	},

	// OpenAI Compatible Providers
	"openai": {
		Name:         "OpenAI",
		BaseURL:      "https://api.openai.com/v1",
		DefaultModel: "gpt-4",
	},
	"deepseek": {
		Name:         "DeepSeek",
		BaseURL:      "https://api.deepseek.com/v1",
		DefaultModel: "deepseek-chat",
	},
	"qwen": {
		Name:         "Qwen",
		BaseURL:      "https://dashscope.aliyuncs.com/compatible-mode/v1",
		DefaultModel: "qwen-plus",
	},
	"moonshot": {
		Name:         "Moonshot",
		BaseURL:      "https://api.moonshot.cn/v1",
		DefaultModel: "moonshot-v1-8k",
	},
	"glm": {
		Name:         "GLM",
		BaseURL:      "https://open.bigmodel.cn/api/paas/v4",
		DefaultModel: "glm-4",
	},
	"xai": {
		Name:         "XAI",
		BaseURL:      "https://api.x.ai/v1",
		DefaultModel: "grok-beta",
	},
	"minimax": {
		Name:         "MiniMax",
		BaseURL:      "https://api.minimax.chat/v1",
		DefaultModel: "abab6.5-chat",
	},
	"baichuan": {
		Name:         "Baichuan",
		BaseURL:      "https://api.baichuan-ai.com/v1",
		DefaultModel: "Baichuan2-Turbo",
	},
	"01ai": {
		Name:         "01.AI",
		BaseURL:      "https://api.01.ai/v1",
		DefaultModel: "yi-large",
	},
	"groq": {
		Name:         "Groq",
		BaseURL:      "https://api.groq.com/openai/v1",
		DefaultModel: "llama-3.1-70b-versatile",
	},
	"together": {
		Name:         "Together AI",
		BaseURL:      "https://api.together.xyz/v1",
		DefaultModel: "meta-llama/Llama-3-70b-chat-hf",
	},
	"siliconflow": {
		Name:         "SiliconFlow",
		BaseURL:      "https://api.siliconflow.cn/v1",
		DefaultModel: "deepseek-ai/DeepSeek-V2.5",
	},

	// Custom provider for user-defined endpoints
	"custom": {
		Name:         "Custom",
		BaseURL:      "",
		DefaultModel: "",
	},
}
