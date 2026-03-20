package ollama

import (
	"net/http"
	"net/url"
	"time"

	"github.com/ollama/ollama/api"
)

type Provider struct {
	client *api.Client
}

func NewProvider(baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		u, _ = url.Parse("http://localhost:11434")
	}
	client := api.NewClient(u, &http.Client{
		Timeout: 120 * time.Second,
	})
	return &Provider{client: client}
}
