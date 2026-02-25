package skills

import "net/http"

type installer struct {
	workspace  string
	client     *http.Client
	registries []SkillRegistry
}
type SkillRegistry struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}
