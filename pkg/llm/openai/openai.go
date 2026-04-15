package openai

import (
	"strings"

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
	var opts []option.RequestOption
	opts = append(opts, option.WithAPIKey(apiKey), option.WithBaseURL(baseURL))

	if strings.Contains(baseURL, "openrouter.ai") {
		opts = append(opts,
			option.WithHeader("HTTP-Referer", constant.HttpReferer),
			option.WithHeader("X-OpenRouter-Title", constant.OpenRouterTitle),
			option.WithHeader("X-OpenRouter-Categories", constant.OpenRouterCategories),
		)
	}

	client := openai.NewClient(
		opts...,
	)
	return &Provider{client: client}
}
