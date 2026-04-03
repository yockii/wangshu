package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg == nil {
		t.Fatal("defaultConfig should not return nil")
	}

	// 检查Agents
	if len(cfg.Agents) == 0 {
		t.Error("defaultConfig should have at least one agent")
	}

	// 检查默认agent
	defaultAgent, ok := cfg.Agents["default"]
	if !ok {
		t.Fatal("defaultConfig should have a 'default' agent")
	}

	if defaultAgent.Workspace == "" {
		t.Error("default agent workspace should not be empty")
	}

	if defaultAgent.Provider == "" {
		t.Error("default agent provider should not be empty")
	}

	if defaultAgent.Model == "" {
		t.Error("default agent model should not be empty")
	}

	// 检查Providers
	if len(cfg.Providers) == 0 {
		t.Error("defaultConfig should have at least one provider")
	}

	// 检查Channels
	if len(cfg.Channels) == 0 {
		t.Error("defaultConfig should have at least one channel")
	}

	// 检查Skill配置
	if cfg.Skill.GlobalPath == "" {
		t.Error("default skill global_path should not be empty")
	}
}

func TestLoadConfig_NotExist(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	configFile = cfgPath

	// 加载不存在的配置文件（应该自动创建）
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig should not return nil")
	}

	// 检查文件是否被创建
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("config file should be created when not exists")
	}
}

func TestLoadConfig_Exist(t *testing.T) {
	// 创建临时目录和配置文件
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	configFile = cfgPath

	// 创建测试配置
	testCfg := &Config{
		Agents: map[string]*AgentConfig{
			"test": {
				Workspace:   "/tmp/workspace",
				Provider:    "testProvider",
				Model:       "test-model",
				Temperature: 0.5,
			},
		},
		Providers: map[string]*ProviderConfig{
			"testProvider": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Channels: map[string]*ChannelConfig{
			"testChannel": {
				Type:    "web",
				Enabled: true,
				Agent:   "test",
			},
		},
	}

	// 保存配置
	err := SaveConfig(testCfg)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// 加载配置
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// 验证加载的配置
	if cfg.Agents["test"].Model != "test-model" {
		t.Error("Loaded config does not match saved config")
	}
}

func TestConfigValidate_Valid(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "myProvider",
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Channels: map[string]*ChannelConfig{
			"webTest": {
				Type:        "web",
				Enabled:     true,
				Agent:       "default",
				HostAddress: "localhost:8080",
				Token:       "test-token",
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Config validation should pass for valid config: %v", err)
	}
}

func TestConfigValidate_MissingWorkspace(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "", // 空工作空间
				Provider:  "myProvider",
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for missing workspace")
	}

	// 验证错误消息包含中文提示和问题数量
	expectedErrMsg := "缺少工作空间配置"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}

	// 验证错误消息格式包含问题数量
	if err != nil && !strings.Contains(err.Error(), "发现") && !strings.Contains(err.Error(), "个问题") {
		t.Errorf("Error message should contain problem count, got: %v", err)
	}
}

func TestConfigValidate_MissingProvider(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "", // 空provider
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for missing provider")
	}

	expectedErrMsg := "缺少Provider配置"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestConfigValidate_MissingAPIKey(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "myProvider",
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "", // 空API key
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for missing API key")
	}
}

func TestConfigValidate_WebChannel_MissingHostAddress(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "myProvider",
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Channels: map[string]*ChannelConfig{
			"webTest": {
				Type:    "web",
				Enabled: true,
				Agent:   "default",
				// 缺少 HostAddress
				Token: "test-token",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for web channel without host_address")
	}
}

func TestConfigValidate_WebChannel_MissingToken(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "myProvider",
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Channels: map[string]*ChannelConfig{
			"webTest": {
				Type:        "web",
				Enabled:     true,
				Agent:       "default",
				HostAddress: "localhost:8080",
				// 缺少 Token
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for web channel without token")
	}
}

func TestConfigValidate_FeishuChannel_MissingAppID(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "myProvider",
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Channels: map[string]*ChannelConfig{
			"feishuTest": {
				Type:    "feishu",
				Enabled: true,
				Agent:   "default",
				// 缺少 AppID
				AppSecret: "test-secret",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for feishu channel without app_id")
	}
}

func TestConfigValidate_FeishuChannel_MissingAppSecret(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "myProvider",
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Channels: map[string]*ChannelConfig{
			"feishuTest": {
				Type:    "feishu",
				Enabled: true,
				Agent:   "default",
				AppID:   "test-app-id",
				// 缺少 AppSecret
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for feishu channel without app_secret")
	}

	expectedErrMsg := "缺少AppSecret配置"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestConfigValidate_MissingModel(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "myProvider",
				Model:     "", // 空model
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for missing model")
	}

	expectedErrMsg := "缺少模型配置"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestConfigValidate_TemperatureOutOfRange(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace:   "/tmp/workspace",
				Provider:    "myProvider",
				Model:       "gpt-4",
				Temperature: 3.0, // 超出范围
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for temperature out of range")
	}

	expectedErrMsg := "Temperature值"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestConfigValidate_TemperatureNegative(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace:   "/tmp/workspace",
				Provider:    "myProvider",
				Model:       "gpt-4",
				Temperature: -0.5, // 负数
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for negative temperature")
	}
}

func TestConfigValidate_ProviderNotFound(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "nonExistentProvider", // 不存在的provider
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for non-existent provider")
	}

	expectedErrMsg := "不存在"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestConfigValidate_AgentNotFound(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "myProvider",
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Channels: map[string]*ChannelConfig{
			"webTest": {
				Type:        "web",
				Enabled:     true,
				Agent:       "nonExistentAgent", // 不存在的agent
				HostAddress: "localhost:8080",
				Token:       "test-token",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for non-existent agent")
	}

	expectedErrMsg := "不存在"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestConfigValidate_InvalidBaseURL(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "myProvider",
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:    "openai",
				APIKey:  "test-api-key",
				BaseURL: "invalid-url", // 无效的URL
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for invalid base URL")
	}

	expectedErrMsg := "BaseURL格式错误"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestConfigValidate_OllamaWithoutAPIKey(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "ollamaProvider",
				Model:     "llama2",
			},
		},
		Providers: map[string]*ProviderConfig{
			"ollamaProvider": {
				Type:   "ollama",
				APIKey: "", // ollama不需要API key
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Ollama provider should work without API key: %v", err)
	}
}

func TestConfigValidate_UnsupportedChannelType(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "myProvider",
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Channels: map[string]*ChannelConfig{
			"unsupportedTest": {
				Type:    "unsupported_type", // 不支持的类型
				Enabled: true,
				Agent:   "default",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail for unsupported channel type")
	}

	expectedErrMsg := "不支持"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestConfigValidate_UnusedProvider(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "/tmp/workspace",
				Provider:  "openaiProvider", // 使用openaiProvider
				Model:     "gpt-4",
			},
		},
		Providers: map[string]*ProviderConfig{
			"openaiProvider": {
				Type:   "openai",
				APIKey: "test-api-key", // 配置完整
			},
			"unusedProvider": { // 未被使用的provider，配置不完整
				Type:   "openai",
				APIKey: "", // 空api key，但因为不被使用，所以不应该报错
			},
			"anotherUnused": { // 另一个未使用的provider
				Type:   "", // 空type，但不应该报错
				APIKey: "",
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validation should pass for unused providers with incomplete config: %v", err)
	}
}

func TestConfigValidate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		Agents: map[string]*AgentConfig{
			"default": {
				Workspace: "",           // 空工作空间
				Provider:  "myProvider", // 引用myProvider，这样它会被验证
				Model:     "",           // 空model
			},
		},
		Providers: map[string]*ProviderConfig{
			"myProvider": {
				Type:   "", // 空type
				APIKey: "", // 空api key
			},
		},
		Channels: map[string]*ChannelConfig{
			"webTest": {
				Type:        "web",
				Enabled:     true,
				Agent:       "", // 空agent
				HostAddress: "", // 空host address
				Token:       "", // 空token
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Config validation should fail with multiple errors")
	}

	// 验证错误消息包含多个问题
	errMsg := err.Error()
	if !strings.Contains(errMsg, "发现") && !strings.Contains(errMsg, "个问题") {
		t.Errorf("Error message should contain problem count, got: %v", errMsg)
	}

	// 验证错误消息包含列表格式
	if !strings.Contains(errMsg, "  - ") {
		t.Errorf("Error message should contain list format, got: %v", errMsg)
	}

	// 验证包含多个具体错误
	expectedSubstrings := []string{
		"缺少工作空间配置",
		"缺少模型配置",
		"缺少类型配置",
		"缺少API密钥",
		"未指定绑定的智能体",
		"缺少主机地址配置",
		"缺少访问令牌配置",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(errMsg, expected) {
			t.Errorf("Error message should contain '%s', got: %v", expected, errMsg)
		}
	}
}
