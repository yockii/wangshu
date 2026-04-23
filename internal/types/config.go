package types

type BaseConfig interface {
	GetID() string
	SetID(string)
}

type McpConfig struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	Cwd     string            `json:"cwd,omitempty"`
	// 通信协议，默认stdio，以后可能扩展到http、sse等
	TransportType string            `json:"transport_type,omitempty"` // 通信协议，默认stdio，以后可能扩展到http、sse等
	URL           string            `json:"url,omitempty"`            // 通信地址，用于http、sse等
	Headers       map[string]string `json:"headers,omitempty"`        // 自定义HTTP请求头，用于http、sse等
}

func (m *McpConfig) GetID() string {
	return m.ID
}

func (m *McpConfig) SetID(id string) {
	m.ID = id
}

type SkillConfig struct {
	ID         string `json:"id"`
	GlobalPath string `json:"global_path"`
}

func (s *SkillConfig) GetID() string {
	return s.ID
}

func (s *SkillConfig) SetID(id string) {
	s.ID = id
}

type BrowserConfig struct {
	ID      string `json:"id"`
	DataDir string `json:"data_dir"` // 浏览器用户数据目录，用于持久化cookies、登录状态等
}

func (b *BrowserConfig) GetID() string {
	return b.ID
}

func (b *BrowserConfig) SetID(id string) {
	b.ID = id
}

type Live2DConfig struct {
	ID        string `json:"id"`
	Enabled   bool   `json:"enabled"`
	ModelDir  string `json:"model_dir"`
	ModelName string `json:"model_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
}

func (l *Live2DConfig) GetID() string {
	return l.ID
}

func (l *Live2DConfig) SetID(id string) {
	l.ID = id
}

type AgentConfig struct {
	ID                     string  `json:"id"`
	AgentName              string  `json:"agent_name"`
	Workspace              string  `json:"workspace"`
	Provider               string  `json:"provider"`
	Model                  string  `json:"model"`
	Temperature            float64 `json:"temperature"`
	MaxTokens              int64   `json:"max_tokens"`
	EnableImageRecognition bool    `json:"enable_image_recognition"`
	// 每日0点或配置的时间进行记忆整理
	MemoryOrganizeTime string `json:"memory_organize_time"`
}

func (a *AgentConfig) GetID() string {
	return a.ID
}

func (a *AgentConfig) SetID(id string) {
	a.ID = id
}

type ProviderConfig struct {
	ID           string `json:"id"`
	ProviderName string `json:"provider_name"`
	Type         string `json:"type"` // openai/anthropic/ollama/...
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url,omitempty"`
}

func (p *ProviderConfig) GetID() string {
	return p.ID
}

func (p *ProviderConfig) SetID(id string) {
	p.ID = id
}

type ChannelConfig struct {
	ID          string `json:"id"`
	ChannelName string `json:"channel_name"`
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	Agent       string `json:"agent"`
	// feishu
	AppID     string `json:"app_id,omitempty"`
	AppSecret string `json:"app_secret,omitempty"`
	// web
	HostAddress string `json:"host_address,omitempty"`
	Token       string `json:"token,omitempty"`
	// wechat ilink
	CredPath string `json:"cred_path,omitempty"` // 凭证存储路径，默认 ~/.wechatbot/{name}_credentials.json
}

func (c *ChannelConfig) GetID() string {
	return c.ID
}

func (c *ChannelConfig) SetID(id string) {
	c.ID = id
}
