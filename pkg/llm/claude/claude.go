package claude

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/yockii/wangshu/pkg/constant"
)

// Provider Anthropic Claude Provider实现
type Provider struct {
	client anthropic.Client
}

// NewProvider 创建一个新的Claude Provider
func NewProvider(apiKey, baseURL string) *Provider {
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithHeader("HTTP-Referer", constant.HttpReferer),
		option.WithHeader("X-OpenRouter-Title", constant.OpenRouterTitle),
		option.WithHeader("X-OpenRouter-Categories", constant.OpenRouterCategories),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	return &Provider{client: anthropic.NewClient(opts...)}
}
