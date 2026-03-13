package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yockii/wangshu/internal/config"
)

type wizardStep int

const (
	stepProviderType wizardStep = iota
	stepAPIKey
	stepBaseURL
	stepModel
	stepWorkspace
	stepSkillPath
	stepBrowserPath
	stepConfirm
)

type wizardModel struct {
	step         wizardStep
	width        int
	height       int
	input        textinput.Model
	providerType string
	apiKey       string
	baseURL      string
	model        string
	workspace    string
	skillPath    string
	browserPath  string
	selectIdx    int
	err          string
}

var (
	wizardTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8be9fd")).
				Bold(true).
				MarginBottom(2)

	wizardPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f8f8f2")).
				MarginBottom(1)

	wizardInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f8f8f2")).
				Background(lipgloss.Color("#44475a")).
				Padding(0, 1).
				MarginTop(1)

	wizardOptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6272a4")).
				PaddingLeft(2)

	wizardOptionActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fafafa")).
				Background(lipgloss.Color("#8B5CF6")).
				PaddingLeft(2).
				Bold(true)

	wizardErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ff5555")).
				MarginTop(1)

	wizardHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272a4")).
			Italic(true).
			MarginTop(2)

	wizardSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffb86c")).
				Bold(true).
				MarginTop(1)
)

func newWizardModel() wizardModel {
	ti := textinput.New()
	ti.Focus()
	ti.Width = 50

	return wizardModel{
		step:      stepProviderType,
		input:     ti,
		selectIdx: 0,
	}
}

func (m wizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		m.err = ""
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit

		case "up":
			if m.step == stepProviderType && m.selectIdx > 0 {
				m.selectIdx--
			}

		case "down":
			if m.step == stepProviderType && m.selectIdx < 2 {
				m.selectIdx++
			}

		case "enter":
			switch m.step {
			case stepProviderType:
				providerTypes := []string{"openai", "anthropic", "ollama"}
				m.providerType = providerTypes[m.selectIdx]
				m.step = stepAPIKey
				m.input.SetValue("")
				m.input.Placeholder = "请输入 API Key"
				if m.providerType == "ollama" {
					m.input.Placeholder = "可选，Ollama 通常不需要 API Key"
				}
			case stepAPIKey:
				m.apiKey = m.input.Value()
				m.step = stepBaseURL
				m.input.SetValue("")
				m.input.Placeholder = "可选，留空使用默认地址"
			case stepBaseURL:
				m.baseURL = m.input.Value()
				m.step = stepModel
				m.input.SetValue("")
				m.input.Placeholder = "例如: gpt-4o, claude-3-5-sonnet-20241022"
			case stepModel:
				m.model = m.input.Value()
				if m.model == "" {
					m.err = "模型名称不能为空"
					return m, nil
				}
				m.step = stepWorkspace
				m.input.SetValue("")
				m.input.Placeholder = "留空使用默认: ~/.wangshu/workspace"
			case stepWorkspace:
				m.workspace = m.input.Value()
				m.step = stepSkillPath
				m.input.SetValue("")
				m.input.Placeholder = "留空使用默认: ~/.wangshu/skills"
			case stepSkillPath:
				m.skillPath = m.input.Value()
				m.step = stepBrowserPath
				m.input.SetValue("")
				m.input.Placeholder = "留空使用默认: ~/.wangshu/browser_profile"
			case stepBrowserPath:
				m.browserPath = m.input.Value()
				m.step = stepConfirm
			case stepConfirm:
				m.saveConfig()
				return m, tea.Quit
			}

		case "esc":
			if m.step > stepProviderType {
				m.step--
				switch m.step {
				case stepProviderType:
					providerTypes := []string{"openai", "anthropic", "ollama"}
					for i, pt := range providerTypes {
						if pt == m.providerType {
							m.selectIdx = i
							break
						}
					}
				case stepAPIKey:
					m.input.SetValue(m.apiKey)
				case stepBaseURL:
					m.input.SetValue(m.baseURL)
				case stepModel:
					m.input.SetValue(m.model)
				case stepWorkspace:
					m.input.SetValue(m.workspace)
				case stepSkillPath:
					m.input.SetValue(m.skillPath)
				case stepBrowserPath:
					m.input.SetValue(m.browserPath)
				}
			}

		default:
			if m.step >= stepAPIKey && m.step <= stepBrowserPath {
				m.input, cmd = m.input.Update(msg)
			}
		}
	}

	return m, cmd
}

func (m *wizardModel) saveConfig() {
	prov := config.DefaultCfg.Providers["default"]
	if prov == nil {
		prov = &config.ProviderConfig{}
		config.DefaultCfg.Providers["default"] = prov
	}
	prov.Type = m.providerType
	prov.APIKey = m.apiKey
	if m.baseURL != "" {
		prov.BaseURL = m.baseURL
	} else if m.providerType == "ollama" {
		prov.BaseURL = "http://localhost:11434"
	}

	agent := config.DefaultCfg.Agents["default"]
	if agent == nil {
		agent = &config.AgentConfig{}
		config.DefaultCfg.Agents["default"] = agent
	}
	agent.Provider = "default"
	agent.Model = m.model
	if m.workspace != "" {
		agent.Workspace = m.workspace
	} else if agent.Workspace == "" {
		agent.Workspace = "~/.wangshu/workspace"
	}

	if m.skillPath != "" {
		config.DefaultCfg.Skill.GlobalPath = m.skillPath
	} else if config.DefaultCfg.Skill.GlobalPath == "" {
		config.DefaultCfg.Skill.GlobalPath = "~/.wangshu/skills"
	}

	if m.browserPath != "" {
		config.DefaultCfg.Browser.DataDir = m.browserPath
	} else if config.DefaultCfg.Browser.DataDir == "" {
		config.DefaultCfg.Browser.DataDir = "~/.wangshu/browser_profile"
	}
}

func (m wizardModel) View() string {
	var sb strings.Builder

	sb.WriteString(wizardTitleStyle.Render("🚀 欢迎使用 WangShu 配置向导"))
	sb.WriteString("\n\n")

	switch m.step {
	case stepProviderType:
		sb.WriteString(wizardPromptStyle.Render("请选择大模型服务提供商："))
		sb.WriteString("\n")
		providerTypes := []string{"📡 OpenAI", "🤖 Anthropic (Claude)", "🏠 Ollama (本地)"}
		for i, pt := range providerTypes {
			style := wizardOptionStyle
			if i == m.selectIdx {
				style = wizardOptionActiveStyle
			}
			sb.WriteString(style.Render(pt))
			sb.WriteString("\n")
		}
		sb.WriteString(wizardHelpStyle.Render("[↑↓选择] [Enter确认]"))

	case stepAPIKey:
		sb.WriteString(wizardPromptStyle.Render(fmt.Sprintf("请输入 %s 的 API Key：", m.providerType)))
		sb.WriteString("\n")
		sb.WriteString(wizardInputStyle.Render(m.input.View()))
		sb.WriteString("\n")
		if m.providerType == "ollama" {
			sb.WriteString(wizardHelpStyle.Render("Ollama 本地运行通常不需要 API Key，可直接按 Enter 跳过"))
		} else {
			sb.WriteString(wizardHelpStyle.Render("[输入 API Key] [Enter确认] [Esc返回]"))
		}

	case stepBaseURL:
		sb.WriteString(wizardPromptStyle.Render("请输入 Base URL（可选）："))
		sb.WriteString("\n")
		sb.WriteString(wizardInputStyle.Render(m.input.View()))
		sb.WriteString("\n")
		var defaultURL string
		switch m.providerType {
		case "openai":
			defaultURL = "默认: https://api.openai.com/v1"
		case "anthropic":
			defaultURL = "默认: https://api.anthropic.com/v1"
		case "ollama":
			defaultURL = "默认: http://localhost:11434"
		}
		sb.WriteString(wizardHelpStyle.Render(defaultURL + " [Enter确认] [Esc返回]"))

	case stepModel:
		sb.WriteString(wizardPromptStyle.Render("请输入要使用的模型名称："))
		sb.WriteString("\n")
		sb.WriteString(wizardInputStyle.Render(m.input.View()))
		sb.WriteString("\n")
		var examples string
		switch m.providerType {
		case "openai":
			examples = "例如: gpt-4o, gpt-4o-mini, gpt-4-turbo"
		case "anthropic":
			examples = "例如: claude-3-5-sonnet-20241022, claude-3-opus-20240229"
		case "ollama":
			examples = "例如: llama3.2, qwen2.5, deepseek-r1"
		}
		sb.WriteString(wizardHelpStyle.Render(examples + " [Enter确认] [Esc返回]"))

	case stepWorkspace:
		sb.WriteString(wizardSectionStyle.Render("📁 高级设置"))
		sb.WriteString("\n\n")
		sb.WriteString(wizardPromptStyle.Render("Agent 工作目录（可选）："))
		sb.WriteString("\n")
		sb.WriteString(wizardInputStyle.Render(m.input.View()))
		sb.WriteString("\n")
		sb.WriteString(wizardHelpStyle.Render("默认: ~/.wangshu/workspace [Enter确认] [Esc返回]"))

	case stepSkillPath:
		sb.WriteString(wizardSectionStyle.Render("📁 高级设置"))
		sb.WriteString("\n\n")
		sb.WriteString(wizardPromptStyle.Render("技能目录（可选）："))
		sb.WriteString("\n")
		sb.WriteString(wizardInputStyle.Render(m.input.View()))
		sb.WriteString("\n")
		sb.WriteString(wizardHelpStyle.Render("默认: ~/.wangshu/skills [Enter确认] [Esc返回]"))

	case stepBrowserPath:
		sb.WriteString(wizardSectionStyle.Render("📁 高级设置"))
		sb.WriteString("\n\n")
		sb.WriteString(wizardPromptStyle.Render("浏览器数据目录（可选）："))
		sb.WriteString("\n")
		sb.WriteString(wizardInputStyle.Render(m.input.View()))
		sb.WriteString("\n")
		sb.WriteString(wizardHelpStyle.Render("默认: ~/.wangshu/browser_profile [Enter确认] [Esc返回]"))

	case stepConfirm:
		sb.WriteString(wizardPromptStyle.Render("配置完成！确认保存？"))
		sb.WriteString("\n\n")
		sb.WriteString(wizardSectionStyle.Render("基础配置"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  Provider:  %s\n", m.providerType))
		sb.WriteString(fmt.Sprintf("  API Key:   %s\n", maskAPIKey(m.apiKey)))
		if m.baseURL != "" {
			sb.WriteString(fmt.Sprintf("  Base URL:  %s\n", m.baseURL))
		} else {
			sb.WriteString("  Base URL:  (使用默认)\n")
		}
		sb.WriteString(fmt.Sprintf("  Model:     %s\n", m.model))
		sb.WriteString("\n")
		sb.WriteString(wizardSectionStyle.Render("高级设置"))
		sb.WriteString("\n")
		if m.workspace != "" {
			sb.WriteString(fmt.Sprintf("  工作目录:   %s\n", m.workspace))
		} else {
			sb.WriteString("  工作目录:   ~/.wangshu/workspace (默认)\n")
		}
		if m.skillPath != "" {
			sb.WriteString(fmt.Sprintf("  技能目录:   %s\n", m.skillPath))
		} else {
			sb.WriteString("  技能目录:   ~/.wangshu/skills (默认)\n")
		}
		if m.browserPath != "" {
			sb.WriteString(fmt.Sprintf("  浏览器目录: %s\n", m.browserPath))
		} else {
			sb.WriteString("  浏览器目录: ~/.wangshu/browser_profile (默认)\n")
		}
		sb.WriteString("\n")
		sb.WriteString(wizardHelpStyle.Render("[Enter保存并启动] [Esc返回修改]"))
	}

	if m.err != "" {
		sb.WriteString("\n")
		sb.WriteString(wizardErrorStyle.Render("❌ " + m.err))
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2, 4).
		Render(sb.String())
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		if key == "" {
			return "(未设置)"
		}
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
