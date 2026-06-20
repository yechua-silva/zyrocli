package opencode

// Provider represents an AI model provider with its available models.
type Provider struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Models []Model `json:"models"`
}

// Model represents a single AI model offered by a provider.
type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// KnownProviders returns the curated list of known providers and their models.
// This is a static list; it does not make HTTP calls.
func KnownProviders() []Provider {
	return []Provider{
		{
			ID: "opencode-go", Name: "OpenCode Go",
			Models: []Model{
				{ID: "deepseek-v4-flash", Name: "DeepSeek V4 Flash"},
				{ID: "deepseek-v4-pro", Name: "DeepSeek V4 Pro"},
				{ID: "mimo-v2.5", Name: "Mimo V2.5"},
				{ID: "mimo-v2.5-pro", Name: "Mimo V2.5 Pro"},
				{ID: "qwen3.7-max", Name: "Qwen 3.7 Max"},
				{ID: "minimax-m3", Name: "MiniMax M3"},
				{ID: "kimi-k2.6", Name: "Kimi K2.6"},
			},
		},
		{
			ID: "opencode", Name: "OpenCode Free",
			Models: []Model{
				{ID: "deepseek-v4-flash-free", Name: "DeepSeek V4 Flash Free"},
				{ID: "mimo-v2.5-free", Name: "Mimo V2.5 Free"},
				{ID: "nemotron-3-super-free", Name: "Nemotron 3 Super Free"},
				{ID: "minimax-m3-free", Name: "MiniMax M3 Free"},
			},
		},
		{
			ID: "google", Name: "Google",
			Models: []Model{
				{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash"},
				{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro"},
			},
		},
		{
			ID: "groq", Name: "Groq",
			Models: []Model{
				{ID: "meta-llama/llama-4-scout-17b-16e-instruct", Name: "Llama 4 Scout 17B"},
			},
		},
		{
			ID: "openrouter", Name: "OpenRouter",
			Models: []Model{
				{ID: "qwen/qwen3-coder:free", Name: "Qwen 3 Coder (Free)"},
			},
		},
		{
			ID: "cerebras", Name: "Cerebras",
			Models: []Model{
				{ID: "gpt-oss-120b", Name: "GPT-OSS 120B"},
			},
		},
		{
			ID: "nvidia", Name: "NVIDIA",
			Models: []Model{
				{ID: "meta/llama-3.1-8b-instruct", Name: "Llama 3.1 8B Instruct"},
				{ID: "meta/llama-3.1-70b-instruct", Name: "Llama 3.1 70B Instruct"},
				{ID: "mistralai/mistral-7b-instruct-v0.3", Name: "Mistral 7B v0.3"},
				{ID: "google/gemma-2-27b-it", Name: "Gemma 2 27B IT"},
				{ID: "microsoft/phi-3-mini-128k-instruct", Name: "Phi-3 Mini 128K"},
				{ID: "nvidia/llama-3.1-nemotron-70b-instruct", Name: "Nemotron 70B"},
				{ID: "nvidia/llama-3.1-nemotron-mini-4b-instruct", Name: "Nemotron Mini 4B"},
			},
		},
		{
			ID: "anthropic", Name: "Anthropic",
			Models: []Model{
				{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6"},
				{ID: "claude-opus-4-20250514", Name: "Claude Opus 4"},
				{ID: "claude-haiku-3-5", Name: "Claude Haiku 3.5"},
			},
		},
	}
}
