package claude

import (
	"strings"

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
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	if strings.Contains(baseURL, "openrouter.ai") {
		opts = append(opts,
			option.WithHeader("HTTP-Referer", constant.HttpReferer),
			option.WithHeader("X-OpenRouter-Title", constant.OpenRouterTitle),
			option.WithHeader("X-OpenRouter-Categories", constant.OpenRouterCategories),
		)
	}

	return &Provider{client: anthropic.NewClient(opts...)}
}
