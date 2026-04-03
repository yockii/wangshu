package openai

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/yockii/wangshu/pkg/constant"
)

// Provider OpenAI Provider实现
type Provider struct {
	client openai.Client
}

// NewProvider 创建一个新的OpenAI Provider
func NewProvider(apiKey, baseURL string) *Provider {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
		option.WithHeader("HTTP-Referer", constant.HttpReferer),
		option.WithHeader("X-OpenRouter-Title", constant.OpenRouterTitle),
		option.WithHeader("X-OpenRouter-Categories", constant.OpenRouterCategories),
	)
	return &Provider{client: client}
}
