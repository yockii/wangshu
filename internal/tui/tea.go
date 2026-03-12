package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yockii/wangshu/internal/config"
)

var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fafafa")).
			Background(lipgloss.Color("#ff79c6")).
			Padding(0, 1).
			MarginBottom(1).
			Bold(true)

	itemStyle = lipgloss.NewStyle().PaddingLeft(4)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#8B5CF6")).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Italic(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8be9fd")).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f8f8f2")).
			Background(lipgloss.Color("#44475a")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5555")).
			MarginLeft(2)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50fa7b")).
			Bold(true).
			MarginTop(1)
)

type mainItem struct {
	title, desc string
}

func (i mainItem) Title() string       { return i.title }
func (i mainItem) FilterValue() string { return i.title }
func (i mainItem) Description() string { return i.desc }

type subItem struct {
	title, desc string
	key         string
}

func (i subItem) Title() string       { return i.title }
func (i subItem) FilterValue() string { return i.title }
func (i subItem) Description() string { return i.desc }

type state int

const (
	mainMenuState state = iota
	selectProviderState
	editProviderState
	selectAgentState
	editAgentState
	selectChannelState
	editChannelState
	doneState
)

type model struct {
	state  state
	width  int
	height int

	mainList list.Model
	subList  list.Model

	currentKey  string
	currentType string

	formFields   map[string]string
	formErrors   map[string]string
	boolFields   map[string]bool
	selectFields map[string][]string
	inputs       map[string]textinput.Model
	currentField int
	fieldOrder   []string
}

func initialModel() model {
	items := []list.Item{
		mainItem{title: "🔧 Providers", desc: "配置大模型服务提供商"},
		mainItem{title: "🤖 Agents", desc: "配置智能 Agent"},
		mainItem{title: "💬 Channels", desc: "配置消息渠道"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Copy().Foreground(lipgloss.Color("#626262"))
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Copy().Foreground(lipgloss.Color("#626262"))

	m := model{
		state:        mainMenuState,
		mainList:     list.New(items, delegate, 0, 0),
		formFields:   make(map[string]string),
		formErrors:   make(map[string]string),
		boolFields:   make(map[string]bool),
		selectFields: make(map[string][]string),
		inputs:       make(map[string]textinput.Model),
	}

	m.mainList.Title = "请选择要配置的部分"
	m.mainList.Styles.Title = titleStyle
	m.mainList.Styles.HelpStyle = helpStyle
	m.mainList.SetFilteringEnabled(false)
	m.mainList.Help.Styles.ShortDesc = helpStyle
	m.mainList.Help.Styles.FullDesc = helpStyle
	m.mainList.KeyMap.Quit = key.NewBinding(
		key.WithKeys("ctrl+q"),
		key.WithHelp("ctrl+q", "退出"),
	)
	m.mainList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("f10"),
				key.WithHelp("f10", "保存并退出"),
			),
		}
	}
	m.mainList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("f10"),
				key.WithHelp("f10", "保存退出"),
			),
		}
	}

	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if winMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = winMsg.Width
		m.height = winMsg.Height
		listHeight := m.height - 4
		switch m.state {
		case mainMenuState:
			m.mainList.SetSize(m.width, listHeight)
		case selectProviderState, selectAgentState, selectChannelState:
			m.subList.SetSize(m.width, listHeight)
		}
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+q":
			return m, tea.Quit
		case "f10":
			if m.state == mainMenuState {
				m.state = doneState
				return m, tea.Quit
			}
		}
	}

	switch m.state {
	case mainMenuState:
		return m.updateMainMenu(msg)
	case selectProviderState:
		return m.updateSelectProvider(msg)
	case editProviderState:
		return m.updateEditProvider(msg)
	case selectAgentState:
		return m.updateSelectAgent(msg)
	case editAgentState:
		return m.updateEditAgent(msg)
	case selectChannelState:
		return m.updateSelectChannel(msg)
	case editChannelState:
		return m.updateEditChannel(msg)
	case doneState:
		return m, tea.Quit
	}

	return m, nil
}

func (m model) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "q":
			return m, nil
		case "enter":
			m.mainList, cmd = m.mainList.Update(msg)
			if i, ok := m.mainList.SelectedItem().(mainItem); ok {
				switch i.title {
				case "🔧 Providers":
					m.state = selectProviderState
					m.subList = m.buildProviderList()
					listHeight := m.height - 4
					if listHeight > 0 {
						m.subList.SetSize(m.width, listHeight)
					}
				case "🤖 Agents":
					m.state = selectAgentState
					m.subList = m.buildAgentList()
					listHeight := m.height - 4
					if listHeight > 0 {
						m.subList.SetSize(m.width, listHeight)
					}
				case "💬 Channels":
					m.state = selectChannelState
					m.subList = m.buildChannelList()
					listHeight := m.height - 4
					if listHeight > 0 {
						m.subList.SetSize(m.width, listHeight)
					}
				}
				return m, cmd
			}
		case "esc":
			return m, nil
		default:
			m.mainList, cmd = m.mainList.Update(msg)
		}
	} else {
		m.mainList, cmd = m.mainList.Update(msg)
	}

	return m, cmd
}

func (m model) updateSelectProvider(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			m.subList, cmd = m.subList.Update(msg)
			if i, ok := m.subList.SelectedItem().(subItem); ok {
				m.currentKey = i.key
				m.currentType = "provider"
				m = m.loadProviderForm(i.key)
				m.state = editProviderState
				return m, cmd
			}
		case "esc":
			m.state = mainMenuState
			return m, nil
		default:
			m.subList, cmd = m.subList.Update(msg)
		}
	} else {
		m.subList, cmd = m.subList.Update(msg)
	}

	return m, cmd
}

func (m model) updateSelectAgent(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			m.subList, cmd = m.subList.Update(msg)
			if i, ok := m.subList.SelectedItem().(subItem); ok {
				m.currentKey = i.key
				m.currentType = "agent"
				m = m.loadAgentForm(i.key)
				m.state = editAgentState
				return m, cmd
			}
		case "esc":
			m.state = mainMenuState
			return m, nil
		default:
			m.subList, cmd = m.subList.Update(msg)
		}
	} else {
		m.subList, cmd = m.subList.Update(msg)
	}

	return m, cmd
}

func (m model) updateSelectChannel(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			m.subList, cmd = m.subList.Update(msg)
			if i, ok := m.subList.SelectedItem().(subItem); ok {
				m.currentKey = i.key
				m.currentType = "channel"
				m = m.loadChannelForm(i.key)
				m.state = editChannelState
				return m, cmd
			}
		case "esc":
			m.state = mainMenuState
			return m, nil
		default:
			m.subList, cmd = m.subList.Update(msg)
		}
	} else {
		m.subList, cmd = m.subList.Update(msg)
	}

	return m, cmd
}

func (m model) updateEditProvider(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up":
			if m.currentField > 0 {
				m.currentField--
				m.focusCurrentField()
			}
		case "down":
			if m.currentField < len(m.fieldOrder)-1 {
				m.currentField++
				m.focusCurrentField()
			}
		case "left":
			currentField := m.fieldOrder[m.currentField]
			if options, isSelect := m.selectFields[currentField]; isSelect && len(options) > 0 {
				currentIdx := -1
				for i, opt := range options {
					if opt == m.formFields[currentField] {
						currentIdx = i
						break
					}
				}
				if currentIdx > 0 {
					m.formFields[currentField] = options[currentIdx-1]
				} else if currentIdx == 0 {
					m.formFields[currentField] = options[len(options)-1]
				} else if len(options) > 0 {
					m.formFields[currentField] = options[0]
				}
			}
		case "right":
			currentField := m.fieldOrder[m.currentField]
			if options, isSelect := m.selectFields[currentField]; isSelect && len(options) > 0 {
				currentIdx := -1
				for i, opt := range options {
					if opt == m.formFields[currentField] {
						currentIdx = i
						break
					}
				}
				if currentIdx >= 0 && currentIdx < len(options)-1 {
					m.formFields[currentField] = options[currentIdx+1]
				} else if currentIdx == len(options)-1 {
					m.formFields[currentField] = options[0]
				} else if len(options) > 0 {
					m.formFields[currentField] = options[0]
				}
			}
		case "enter":
			if m.validateProviderForm() {
				m.saveProviderForm()
				m.state = selectProviderState
				return m, nil
			}
		case "esc":
			m.state = selectProviderState
			return m, nil
		case "f10":
			if m.validateProviderForm() {
				m.saveProviderForm()
				m.state = doneState
				return m, tea.Quit
			}
		}
	}

	currentField := m.fieldOrder[m.currentField]
	if _, isSelect := m.selectFields[currentField]; !isSelect {
		if input, ok := m.inputs[currentField]; ok {
			m.inputs[currentField], cmd = input.Update(msg)
			m.formFields[currentField] = m.inputs[currentField].Value()
		}
	}

	return m, cmd
}

func (m *model) focusCurrentField() {
	for i, field := range m.fieldOrder {
		if input, ok := m.inputs[field]; ok {
			if i == m.currentField {
				input.Focus()
			} else {
				input.Blur()
			}
			m.inputs[field] = input
		}
	}
}

func (m model) updateEditAgent(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up":
			if m.currentField > 0 {
				m.currentField--
				m.focusCurrentField()
			}
		case "down":
			if m.currentField < len(m.fieldOrder)-1 {
				m.currentField++
				m.focusCurrentField()
			}
		case "left":
			currentField := m.fieldOrder[m.currentField]
			if options, isSelect := m.selectFields[currentField]; isSelect && len(options) > 0 {
				currentIdx := -1
				for i, opt := range options {
					if opt == m.formFields[currentField] {
						currentIdx = i
						break
					}
				}
				if currentIdx > 0 {
					m.formFields[currentField] = options[currentIdx-1]
				} else if currentIdx == 0 {
					m.formFields[currentField] = options[len(options)-1]
				} else if len(options) > 0 {
					m.formFields[currentField] = options[0]
				}
			}
		case "right":
			currentField := m.fieldOrder[m.currentField]
			if options, isSelect := m.selectFields[currentField]; isSelect && len(options) > 0 {
				currentIdx := -1
				for i, opt := range options {
					if opt == m.formFields[currentField] {
						currentIdx = i
						break
					}
				}
				if currentIdx >= 0 && currentIdx < len(options)-1 {
					m.formFields[currentField] = options[currentIdx+1]
				} else if currentIdx == len(options)-1 {
					m.formFields[currentField] = options[0]
				} else if len(options) > 0 {
					m.formFields[currentField] = options[0]
				}
			}
		case "enter":
			if m.validateAgentForm() {
				m.saveAgentForm()
				m.state = selectAgentState
				return m, nil
			}
		case "esc":
			m.state = selectAgentState
			return m, nil
		case "f10":
			if m.validateAgentForm() {
				m.saveAgentForm()
				m.state = doneState
				return m, tea.Quit
			}
		}
	}

	currentField := m.fieldOrder[m.currentField]
	if _, isSelect := m.selectFields[currentField]; !isSelect {
		if input, ok := m.inputs[currentField]; ok {
			m.inputs[currentField], cmd = input.Update(msg)
			m.formFields[currentField] = m.inputs[currentField].Value()
		}
	}

	return m, cmd
}

func (m model) updateEditChannel(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up":
			if m.currentField > 0 {
				m.currentField--
				m.focusCurrentField()
			}
		case "down":
			if m.currentField < len(m.fieldOrder)-1 {
				m.currentField++
				m.focusCurrentField()
			}
		case "left":
			currentField := m.fieldOrder[m.currentField]
			if _, isBool := m.boolFields[currentField]; isBool {
				m.boolFields[currentField] = !m.boolFields[currentField]
			} else if options, isSelect := m.selectFields[currentField]; isSelect && len(options) > 0 {
				currentIdx := -1
				for i, opt := range options {
					if opt == m.formFields[currentField] {
						currentIdx = i
						break
					}
				}
				if currentIdx > 0 {
					m.formFields[currentField] = options[currentIdx-1]
				} else if currentIdx == 0 {
					m.formFields[currentField] = options[len(options)-1]
				} else if len(options) > 0 {
					m.formFields[currentField] = options[0]
				}
			}
		case "right":
			currentField := m.fieldOrder[m.currentField]
			if _, isBool := m.boolFields[currentField]; isBool {
				m.boolFields[currentField] = !m.boolFields[currentField]
			} else if options, isSelect := m.selectFields[currentField]; isSelect && len(options) > 0 {
				currentIdx := -1
				for i, opt := range options {
					if opt == m.formFields[currentField] {
						currentIdx = i
						break
					}
				}
				if currentIdx >= 0 && currentIdx < len(options)-1 {
					m.formFields[currentField] = options[currentIdx+1]
				} else if currentIdx == len(options)-1 {
					m.formFields[currentField] = options[0]
				} else if len(options) > 0 {
					m.formFields[currentField] = options[0]
				}
			}
		case " ":
			currentField := m.fieldOrder[m.currentField]
			if _, isBool := m.boolFields[currentField]; isBool {
				m.boolFields[currentField] = !m.boolFields[currentField]
			}
		case "enter":
			if m.validateChannelForm() {
				m.saveChannelForm()
				m.state = selectChannelState
				return m, nil
			}
		case "esc":
			m.state = selectChannelState
			return m, nil
		case "f10":
			if m.validateChannelForm() {
				m.saveChannelForm()
				m.state = doneState
				return m, tea.Quit
			}
		}
	}

	currentField := m.fieldOrder[m.currentField]
	if _, isBool := m.boolFields[currentField]; !isBool {
		if _, isSelect := m.selectFields[currentField]; !isSelect {
			if input, ok := m.inputs[currentField]; ok {
				m.inputs[currentField], cmd = input.Update(msg)
				m.formFields[currentField] = m.inputs[currentField].Value()
			}
		}
	}

	return m, cmd
}

func (m model) buildProviderList() list.Model {
	var items []list.Item
	for key, prov := range config.DefaultCfg.Providers {
		desc := fmt.Sprintf("Type: %s", prov.Type)
		items = append(items, subItem{title: key, desc: desc, key: key})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "选择 Provider"
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = helpStyle
	return l
}

func (m model) buildAgentList() list.Model {
	var items []list.Item
	for key, agent := range config.DefaultCfg.Agents {
		desc := fmt.Sprintf("Model: %s, Provider: %s", agent.Model, agent.Provider)
		items = append(items, subItem{title: key, desc: desc, key: key})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "选择 Agent"
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = helpStyle
	return l
}

func (m model) buildChannelList() list.Model {
	var items []list.Item
	for key, ch := range config.DefaultCfg.Channels {
		status := "禁用"
		if ch.Enabled {
			status = "启用"
		}
		desc := fmt.Sprintf("Type: %s, 状态: %s", ch.Type, status)
		items = append(items, subItem{title: key, desc: desc, key: key})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "选择 Channel"
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = helpStyle
	return l
}

func (m model) loadProviderForm(key string) model {
	prov := config.DefaultCfg.Providers[key]
	m.formFields = make(map[string]string)
	m.formErrors = make(map[string]string)
	m.boolFields = make(map[string]bool)
	m.selectFields = make(map[string][]string)
	m.inputs = make(map[string]textinput.Model)
	m.currentField = 0

	m.fieldOrder = []string{"type", "apiKey", "baseUrl"}

	m.selectFields["type"] = []string{"openai", "anthropic"}
	m.formFields["type"] = prov.Type

	apiKeyInput := textinput.New()
	apiKeyInput.Placeholder = "输入 API Key"
	apiKeyInput.SetValue(prov.APIKey)
	apiKeyInput.EchoMode = textinput.EchoPassword
	apiKeyInput.Focus()
	apiKeyInput.Width = 50
	apiKeyInput.PromptStyle = labelStyle
	apiKeyInput.TextStyle = inputStyle
	apiKeyInput.PlaceholderStyle = helpStyle
	m.inputs["apiKey"] = apiKeyInput
	m.formFields["apiKey"] = prov.APIKey

	baseUrlInput := textinput.New()
	baseUrlInput.Placeholder = "https://api.openai.com/v1"
	baseUrlInput.SetValue(prov.BaseURL)
	baseUrlInput.Width = 50
	baseUrlInput.PromptStyle = labelStyle
	baseUrlInput.TextStyle = inputStyle
	baseUrlInput.PlaceholderStyle = helpStyle
	m.inputs["baseUrl"] = baseUrlInput
	m.formFields["baseUrl"] = prov.BaseURL

	return m
}

func (m model) validateProviderForm() bool {
	m.formErrors = make(map[string]string)

	if m.formFields["type"] == "" {
		m.formErrors["type"] = "类型不能为空"
	}

	if m.formFields["type"] != "ollama" && m.formFields["apiKey"] == "" {
		m.formErrors["apiKey"] = "API Key 不能为空"
	}

	return len(m.formErrors) == 0
}

func (m model) saveProviderForm() {
	if prov, exists := config.DefaultCfg.Providers[m.currentKey]; exists {
		prov.Type = m.formFields["type"]
		prov.APIKey = m.formFields["apiKey"]
		prov.BaseURL = m.formFields["baseUrl"]
	}
}

func (m model) loadAgentForm(key string) model {
	agent := config.DefaultCfg.Agents[key]
	m.formFields = make(map[string]string)
	m.formErrors = make(map[string]string)
	m.boolFields = make(map[string]bool)
	m.selectFields = make(map[string][]string)
	m.inputs = make(map[string]textinput.Model)
	m.currentField = 0

	m.fieldOrder = []string{"provider", "model", "workspace"}

	var providers []string
	for k := range config.DefaultCfg.Providers {
		providers = append(providers, k)
	}
	m.selectFields["provider"] = providers
	m.formFields["provider"] = agent.Provider

	modelInput := textinput.New()
	modelInput.Placeholder = "模型名称"
	modelInput.SetValue(agent.Model)
	modelInput.Focus()
	modelInput.Width = 50
	modelInput.PromptStyle = labelStyle
	modelInput.TextStyle = inputStyle
	modelInput.PlaceholderStyle = helpStyle
	m.inputs["model"] = modelInput
	m.formFields["model"] = agent.Model

	workspaceInput := textinput.New()
	workspaceInput.Placeholder = "~/.wangshu/workspace"
	workspaceInput.SetValue(agent.Workspace)
	workspaceInput.Width = 50
	workspaceInput.PromptStyle = labelStyle
	workspaceInput.TextStyle = inputStyle
	workspaceInput.PlaceholderStyle = helpStyle
	m.inputs["workspace"] = workspaceInput
	m.formFields["workspace"] = agent.Workspace

	return m
}

func (m model) validateAgentForm() bool {
	m.formErrors = make(map[string]string)

	if m.formFields["provider"] == "" {
		m.formErrors["provider"] = "Provider 不能为空"
	}

	if m.formFields["model"] == "" {
		m.formErrors["model"] = "Model 不能为空"
	}

	if m.formFields["workspace"] == "" {
		m.formErrors["workspace"] = "Workspace 不能为空"
	}

	return len(m.formErrors) == 0
}

func (m model) saveAgentForm() {
	if agent, exists := config.DefaultCfg.Agents[m.currentKey]; exists {
		agent.Provider = m.formFields["provider"]
		agent.Model = m.formFields["model"]
		agent.Workspace = m.formFields["workspace"]
	}
}

func (m model) loadChannelForm(key string) model {
	ch := config.DefaultCfg.Channels[key]
	m.formFields = make(map[string]string)
	m.formErrors = make(map[string]string)
	m.boolFields = make(map[string]bool)
	m.selectFields = make(map[string][]string)
	m.inputs = make(map[string]textinput.Model)
	m.currentField = 0

	m.fieldOrder = []string{"type", "agent", "enabled"}

	m.selectFields["type"] = []string{"feishu", "web"}
	m.formFields["type"] = ch.Type

	var agents []string
	for k := range config.DefaultCfg.Agents {
		agents = append(agents, k)
	}
	m.selectFields["agent"] = agents
	m.formFields["agent"] = ch.Agent

	m.boolFields["enabled"] = ch.Enabled

	if ch.Type == "feishu" {
		m.fieldOrder = append(m.fieldOrder, "appId", "appSecret")
		appIdInput := textinput.New()
		appIdInput.Placeholder = "飞书 App ID"
		appIdInput.SetValue(ch.AppID)
		appIdInput.Width = 50
		appIdInput.PromptStyle = labelStyle
		appIdInput.TextStyle = inputStyle
		appIdInput.PlaceholderStyle = helpStyle
		m.inputs["appId"] = appIdInput
		m.formFields["appId"] = ch.AppID

		appSecretInput := textinput.New()
		appSecretInput.Placeholder = "飞书 App Secret"
		appSecretInput.SetValue(ch.AppSecret)
		appSecretInput.EchoMode = textinput.EchoPassword
		appSecretInput.Width = 50
		appSecretInput.PromptStyle = labelStyle
		appSecretInput.TextStyle = inputStyle
		appSecretInput.PlaceholderStyle = helpStyle
		m.inputs["appSecret"] = appSecretInput
		m.formFields["appSecret"] = ch.AppSecret
	} else if ch.Type == "web" {
		m.fieldOrder = append(m.fieldOrder, "hostAddress", "token")
		hostInput := textinput.New()
		hostInput.Placeholder = "localhost:8080"
		hostInput.SetValue(ch.HostAddress)
		hostInput.Width = 50
		hostInput.PromptStyle = labelStyle
		hostInput.TextStyle = inputStyle
		hostInput.PlaceholderStyle = helpStyle
		m.inputs["hostAddress"] = hostInput
		m.formFields["hostAddress"] = ch.HostAddress

		tokenInput := textinput.New()
		tokenInput.Placeholder = "访问令牌"
		tokenInput.SetValue(ch.Token)
		tokenInput.EchoMode = textinput.EchoPassword
		tokenInput.Width = 50
		tokenInput.PromptStyle = labelStyle
		tokenInput.TextStyle = inputStyle
		tokenInput.PlaceholderStyle = helpStyle
		m.inputs["token"] = tokenInput
		m.formFields["token"] = ch.Token
	}

	return m
}

func (m model) validateChannelForm() bool {
	m.formErrors = make(map[string]string)

	if m.formFields["type"] == "" {
		m.formErrors["type"] = "类型不能为空"
	}

	if m.formFields["agent"] == "" {
		m.formErrors["agent"] = "Agent 不能为空"
	}

	if m.formFields["type"] == "feishu" {
		if m.formFields["appId"] == "" {
			m.formErrors["appId"] = "App ID 不能为空"
		}
		if m.formFields["appSecret"] == "" {
			m.formErrors["appSecret"] = "App Secret 不能为空"
		}
	} else if m.formFields["type"] == "web" {
		if m.formFields["hostAddress"] == "" {
			m.formErrors["hostAddress"] = "Host Address 不能为空"
		}
		if m.formFields["token"] == "" {
			m.formErrors["token"] = "Token 不能为空"
		}
	}

	return len(m.formErrors) == 0
}

func (m model) saveChannelForm() {
	if ch, exists := config.DefaultCfg.Channels[m.currentKey]; exists {
		ch.Type = m.formFields["type"]
		ch.Agent = m.formFields["agent"]
		ch.Enabled = m.boolFields["enabled"]
		if ch.Type == "feishu" {
			ch.AppID = m.formFields["appId"]
			ch.AppSecret = m.formFields["appSecret"]
		} else if ch.Type == "web" {
			ch.HostAddress = m.formFields["hostAddress"]
			ch.Token = m.formFields["token"]
		}
	}
}

func (m model) View() string {
	var s string

	switch m.state {
	case mainMenuState:
		s = m.mainList.View()
	case selectProviderState, selectAgentState, selectChannelState:
		if m.height > 0 {
			listHeight := m.height - 4
			if listHeight > 0 && m.width > 0 {
				m.subList.SetSize(m.width, listHeight)
			}
		}
		s = m.subList.View()
	case editProviderState:
		s = m.renderProviderForm()
	case editAgentState:
		s = m.renderAgentForm()
	case editChannelState:
		s = m.renderChannelForm()
	case doneState:
		s = "配置完成！"
	}

	return docStyle.Render(s)
}

func (m model) renderProviderForm() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("编辑 Provider: "+m.currentKey) + "\n\n")

	for i, field := range m.fieldOrder {
		label := ""
		switch field {
		case "type":
			label = "类型"
		case "apiKey":
			label = "API Key"
		case "baseUrl":
			label = "Base URL"
		}

		isCurrentField := i == m.currentField
		fieldStyle := labelStyle
		if !isCurrentField {
			fieldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
		}

		b.WriteString(fieldStyle.Render(label) + "\n")

		if options, isSelect := m.selectFields[field]; isSelect {
			currentValue := m.formFields[field]
			selectStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
			if isCurrentField {
				selectStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#fafafa")).
					Background(lipgloss.Color("#8B5CF6")).
					Padding(0, 1)
			}
			var optionsDisplay []string
			for _, opt := range options {
				if opt == currentValue {
					optionsDisplay = append(optionsDisplay, fmt.Sprintf("[%s]", opt))
				} else {
					optionsDisplay = append(optionsDisplay, fmt.Sprintf(" %s ", opt))
				}
			}
			b.WriteString(selectStyle.Render(strings.Join(optionsDisplay, " ")) + "\n")
		} else {
			input := m.inputs[field]
			b.WriteString(input.View() + "\n")
		}

		if err, ok := m.formErrors[field]; ok {
			b.WriteString(errorStyle.Render("⚠ "+err) + "\n")
		}

		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("↑↓ 切换字段 • ←→ 切换选项 • Enter 保存 • Esc 返回 • F10 保存并退出"))

	return b.String()
}

func (m model) renderAgentForm() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("编辑 Agent: "+m.currentKey) + "\n\n")

	for i, field := range m.fieldOrder {
		label := ""
		switch field {
		case "provider":
			label = "Provider"
		case "model":
			label = "Model"
		case "workspace":
			label = "Workspace"
		}

		isCurrentField := i == m.currentField
		fieldStyle := labelStyle
		if !isCurrentField {
			fieldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
		}

		b.WriteString(fieldStyle.Render(label) + "\n")

		if options, isSelect := m.selectFields[field]; isSelect {
			currentValue := m.formFields[field]
			selectStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
			if isCurrentField {
				selectStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#fafafa")).
					Background(lipgloss.Color("#8B5CF6")).
					Padding(0, 1)
			}
			var optionsDisplay []string
			for _, opt := range options {
				if opt == currentValue {
					optionsDisplay = append(optionsDisplay, fmt.Sprintf("[%s]", opt))
				} else {
					optionsDisplay = append(optionsDisplay, fmt.Sprintf(" %s ", opt))
				}
			}
			b.WriteString(selectStyle.Render(strings.Join(optionsDisplay, " ")) + "\n")
		} else {
			input := m.inputs[field]
			b.WriteString(input.View() + "\n")
		}

		if err, ok := m.formErrors[field]; ok {
			b.WriteString(errorStyle.Render("⚠ "+err) + "\n")
		}

		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("↑↓ 切换字段 • ←→ 切换选项 • Enter 保存 • Esc 返回 • F10 保存并退出"))

	return b.String()
}

func (m model) renderChannelForm() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("编辑 Channel: "+m.currentKey) + "\n\n")

	for i, field := range m.fieldOrder {
		label := ""
		switch field {
		case "type":
			label = "类型"
		case "agent":
			label = "绑定的 Agent"
		case "enabled":
			label = "启用状态"
		case "appId":
			label = "App ID"
		case "appSecret":
			label = "App Secret"
		case "hostAddress":
			label = "Host Address"
		case "token":
			label = "Token"
		}

		isCurrentField := i == m.currentField
		fieldStyle := labelStyle
		if !isCurrentField {
			fieldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
		}

		b.WriteString(fieldStyle.Render(label) + "\n")

		if isBool, ok := m.boolFields[field]; ok {
			checkbox := "[ ]"
			checkboxStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
			if isBool {
				checkbox = "[✓]"
				checkboxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b"))
			}
			if isCurrentField {
				checkboxStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#fafafa")).
					Background(lipgloss.Color("#8B5CF6")).
					Padding(0, 1)
			}
			b.WriteString(checkboxStyle.Render(checkbox) + "\n")
		} else if options, isSelect := m.selectFields[field]; isSelect {
			currentValue := m.formFields[field]
			selectStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
			if isCurrentField {
				selectStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#fafafa")).
					Background(lipgloss.Color("#8B5CF6")).
					Padding(0, 1)
			}
			var optionsDisplay []string
			for _, opt := range options {
				if opt == currentValue {
					optionsDisplay = append(optionsDisplay, fmt.Sprintf("[%s]", opt))
				} else {
					optionsDisplay = append(optionsDisplay, fmt.Sprintf(" %s ", opt))
				}
			}
			b.WriteString(selectStyle.Render(strings.Join(optionsDisplay, " ")) + "\n")
		} else {
			input := m.inputs[field]
			b.WriteString(input.View() + "\n")
		}

		if err, ok := m.formErrors[field]; ok {
			b.WriteString(errorStyle.Render("⚠ "+err) + "\n")
		}

		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("↑↓ 切换字段 • ←→/空格 切换选项 • Enter 保存 • Esc 返回 • F10 保存并退出"))

	return b.String()
}
