package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yockii/wangshu/internal/config"
)

type focusArea int

const (
	focusSidebar focusArea = iota
	focusContent
)

type configPanelModel struct {
	width       int
	height      int
	focus       focusArea
	category    int
	items       []string
	selectedIdx int

	editMode        bool
	editField       int
	editFields      []string
	editFieldType   map[string]string
	inputs          map[string]textinput.Model
	selectOpts      map[string][]string
	selectVal       map[string]string
	boolVal         map[string]bool
	editKey         string
	prevChannelType string
	configChanged   bool
	showSaveConfirm bool
}

var (
	sidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	sidebarActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#8B5CF6")).
				Padding(0, 1)

	contentStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	contentActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#8B5CF6")).
				Padding(0, 1)

	categoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272a4")).
			Padding(0, 1)

	categoryActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fafafa")).
				Background(lipgloss.Color("#8B5CF6")).
				Padding(0, 1).
				Bold(true)

	cfgItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272a4")).
			PaddingLeft(2)

	cfgItemActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fafafa")).
				Background(lipgloss.Color("#8B5CF6")).
				PaddingLeft(2).
				Bold(true)

	cfgLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8be9fd")).
			Bold(true)

	cfgInputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f8f8f2")).
			Background(lipgloss.Color("#44475a")).
			Padding(0, 1)

	cfgHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272a4")).
			Italic(true)
)

func newConfigPanelModel() configPanelModel {
	m := configPanelModel{
		focus:         focusSidebar,
		category:      0,
		editMode:      false,
		editField:     0,
		editFields:    []string{},
		editFieldType: make(map[string]string),
		inputs:        make(map[string]textinput.Model),
		selectOpts:    make(map[string][]string),
		selectVal:     make(map[string]string),
		boolVal:       make(map[string]bool),
	}
	m.loadItems()
	return m
}

func (m configPanelModel) Init() tea.Cmd {
	return nil
}

func (m configPanelModel) Update(msg tea.Msg) (configPanelModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.showSaveConfirm {
			return m.handleSaveConfirm(msg)
		}

		if m.editMode {
			return m.handleEditKeys(msg)
		}

		switch msg.String() {
		case "up":
			if m.focus == focusSidebar {
				if m.category > 0 {
					m.category--
					m.loadItems()
					m.selectedIdx = 0
				}
			} else {
				if m.selectedIdx > 0 {
					m.selectedIdx--
				}
			}
		case "down":
			if m.focus == focusSidebar {
				if m.category < 3 {
					m.category++
					m.loadItems()
					m.selectedIdx = 0
				}
			} else {
				if m.selectedIdx < len(m.items)-1 {
					m.selectedIdx++
				}
			}
		case "enter":
			if m.focus == focusSidebar {
				m.focus = focusContent
				m.selectedIdx = 0
			} else {
				if len(m.items) > 0 {
					m.startEdit()
				}
			}
		case "esc":
			if m.focus == focusContent {
				m.focus = focusSidebar
			}
		case "f10":
			m.showSaveConfirm = true
		}
	}

	return m, cmd
}

func (m configPanelModel) handleSaveConfirm(msg tea.KeyMsg) (configPanelModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.showSaveConfirm = false
		return m, tea.Quit
	case "n", "N", "esc":
		m.showSaveConfirm = false
	}
	return m, nil
}

func (m configPanelModel) handleEditKeys(msg tea.KeyMsg) (configPanelModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.editMode = false
		m.loadItems()
		return m, nil
	case "up":
		if m.editField > 0 {
			m.editField--
			m.focusEditField()
		}
	case "down":
		if m.editField < len(m.editFields)-1 {
			m.editField++
			m.focusEditField()
		}
	case "left":
		m.handleLeftRight(false)
	case "right":
		m.handleLeftRight(true)
	case " ":
		field := m.editFields[m.editField]
		if m.editFieldType[field] == "bool" {
			m.boolVal[field] = !m.boolVal[field]
		}
	case "enter":
		m.saveEdit()
		m.configChanged = true
		m.editMode = false
		m.loadItems()
	default:
		field := m.editFields[m.editField]
		if input, ok := m.inputs[field]; ok {
			m.inputs[field], cmd = input.Update(msg)
		}
	}

	return m, cmd
}

func (m *configPanelModel) handleLeftRight(right bool) {
	field := m.editFields[m.editField]
	switch m.editFieldType[field] {
	case "select":
		opts := m.selectOpts[field]
		if len(opts) == 0 {
			return
		}
		currentIdx := -1
		for i, opt := range opts {
			if opt == m.selectVal[field] {
				currentIdx = i
				break
			}
		}
		if right {
			if currentIdx < len(opts)-1 {
				m.selectVal[field] = opts[currentIdx+1]
			} else {
				m.selectVal[field] = opts[0]
			}
		} else {
			if currentIdx > 0 {
				m.selectVal[field] = opts[currentIdx-1]
			} else {
				m.selectVal[field] = opts[len(opts)-1]
			}
		}
		if field == "type" && m.category == 2 {
			m.updateChannelFields()
		}
	case "bool":
		m.boolVal[field] = !m.boolVal[field]
	}
}

func (m *configPanelModel) loadItems() {
	m.items = []string{}
	switch m.category {
	case 0:
		for k := range config.DefaultCfg.Providers {
			m.items = append(m.items, k)
		}
	case 1:
		for k := range config.DefaultCfg.Agents {
			m.items = append(m.items, k)
		}
	case 2:
		for k := range config.DefaultCfg.Channels {
			m.items = append(m.items, k)
		}
	case 3:
		m.items = []string{"Skills 路径", "Browser 数据目录"}
	}
	if len(m.items) == 0 {
		m.selectedIdx = 0
	} else if m.selectedIdx >= len(m.items) {
		m.selectedIdx = len(m.items) - 1
	}
}

func (m *configPanelModel) startEdit() {
	m.editMode = true
	m.editField = 0
	m.editFields = []string{}
	m.editFieldType = make(map[string]string)
	m.inputs = make(map[string]textinput.Model)
	m.selectOpts = make(map[string][]string)
	m.selectVal = make(map[string]string)
	m.boolVal = make(map[string]bool)

	if len(m.items) == 0 {
		return
	}
	m.editKey = m.items[m.selectedIdx]

	switch m.category {
	case 0:
		m.loadProviderEdit()
	case 1:
		m.loadAgentEdit()
	case 2:
		m.loadChannelEdit()
	case 3:
		m.loadSystemEdit()
	}

	m.focusEditField()
}

func (m *configPanelModel) loadProviderEdit() {
	prov := config.DefaultCfg.Providers[m.editKey]

	m.editFields = []string{"type", "apiKey", "baseUrl"}
	m.editFieldType["type"] = "select"
	m.editFieldType["apiKey"] = "text"
	m.editFieldType["baseUrl"] = "text"

	m.selectOpts["type"] = []string{"openai", "anthropic", "ollama"}
	m.selectVal["type"] = prov.Type

	ti := textinput.New()
	ti.SetValue(prov.APIKey)
	ti.EchoMode = textinput.EchoPassword
	ti.Width = 40
	m.inputs["apiKey"] = ti

	ti2 := textinput.New()
	ti2.SetValue(prov.BaseURL)
	ti2.Width = 40
	m.inputs["baseUrl"] = ti2
}

func (m *configPanelModel) loadAgentEdit() {
	agent := config.DefaultCfg.Agents[m.editKey]

	m.editFields = []string{"provider", "model", "workspace"}
	m.editFieldType["provider"] = "select"
	m.editFieldType["model"] = "text"
	m.editFieldType["workspace"] = "text"

	var providers []string
	for k := range config.DefaultCfg.Providers {
		providers = append(providers, k)
	}
	m.selectOpts["provider"] = providers
	m.selectVal["provider"] = agent.Provider

	ti := textinput.New()
	ti.SetValue(agent.Model)
	ti.Width = 40
	m.inputs["model"] = ti

	ti2 := textinput.New()
	ti2.SetValue(agent.Workspace)
	ti2.Width = 40
	m.inputs["workspace"] = ti2
}

func (m *configPanelModel) loadChannelEdit() {
	ch := config.DefaultCfg.Channels[m.editKey]

	m.editFields = []string{"type", "agent", "enabled"}
	m.editFieldType["type"] = "select"
	m.editFieldType["agent"] = "select"
	m.editFieldType["enabled"] = "bool"

	m.selectOpts["type"] = []string{"feishu", "web"}
	m.selectVal["type"] = ch.Type
	m.prevChannelType = ch.Type

	var agents []string
	for k := range config.DefaultCfg.Agents {
		agents = append(agents, k)
	}
	m.selectOpts["agent"] = agents
	m.selectVal["agent"] = ch.Agent

	m.boolVal["enabled"] = ch.Enabled

	m.addChannelTypeFields()
}

func (m *configPanelModel) addChannelTypeFields() {
	ch := config.DefaultCfg.Channels[m.editKey]
	channelType := m.selectVal["type"]

	if channelType == "feishu" {
		m.editFields = append(m.editFields, "appID", "appSecret")
		m.editFieldType["appID"] = "text"
		m.editFieldType["appSecret"] = "text"

		ti := textinput.New()
		ti.SetValue(ch.AppID)
		ti.Width = 40
		m.inputs["appID"] = ti

		ti2 := textinput.New()
		ti2.SetValue(ch.AppSecret)
		ti2.EchoMode = textinput.EchoPassword
		ti2.Width = 40
		m.inputs["appSecret"] = ti2
	} else if channelType == "web" {
		m.editFields = append(m.editFields, "hostAddress", "token")
		m.editFieldType["hostAddress"] = "text"
		m.editFieldType["token"] = "text"

		ti := textinput.New()
		ti.SetValue(ch.HostAddress)
		ti.Placeholder = "localhost:8080"
		ti.Width = 40
		m.inputs["hostAddress"] = ti

		ti2 := textinput.New()
		ti2.SetValue(ch.Token)
		ti2.Width = 40
		m.inputs["token"] = ti2
	}
}

func (m *configPanelModel) updateChannelFields() {
	ch := config.DefaultCfg.Channels[m.editKey]
	newType := m.selectVal["type"]

	if m.prevChannelType == newType {
		return
	}
	m.prevChannelType = newType

	baseFields := []string{"type", "agent", "enabled"}
	baseTypes := map[string]string{
		"type":    "select",
		"agent":   "select",
		"enabled": "bool",
	}

	newFields := make([]string, len(baseFields))
	copy(newFields, baseFields)
	m.editFieldType = baseTypes

	m.inputs = make(map[string]textinput.Model)

	if newType == "feishu" {
		newFields = append(newFields, "appID", "appSecret")
		m.editFieldType["appID"] = "text"
		m.editFieldType["appSecret"] = "text"

		ti := textinput.New()
		ti.SetValue(ch.AppID)
		ti.Width = 40
		m.inputs["appID"] = ti

		ti2 := textinput.New()
		ti2.SetValue(ch.AppSecret)
		ti2.EchoMode = textinput.EchoPassword
		ti2.Width = 40
		m.inputs["appSecret"] = ti2
	} else if newType == "web" {
		newFields = append(newFields, "hostAddress", "token")
		m.editFieldType["hostAddress"] = "text"
		m.editFieldType["token"] = "text"

		ti := textinput.New()
		ti.SetValue(ch.HostAddress)
		ti.Placeholder = "localhost:8080"
		ti.Width = 40
		m.inputs["hostAddress"] = ti

		ti2 := textinput.New()
		ti2.SetValue(ch.Token)
		ti2.Width = 40
		m.inputs["token"] = ti2
	}

	m.editFields = newFields
	if m.editField >= len(m.editFields) {
		m.editField = len(m.editFields) - 1
	}
	m.focusEditField()
}

func (m *configPanelModel) loadSystemEdit() {
	m.editFields = []string{"skillsPath", "browserDataDir"}
	m.editFieldType["skillsPath"] = "text"
	m.editFieldType["browserDataDir"] = "text"

	ti := textinput.New()
	ti.SetValue(config.DefaultCfg.Skill.GlobalPath)
	ti.Width = 40
	m.inputs["skillsPath"] = ti

	ti2 := textinput.New()
	ti2.SetValue(config.DefaultCfg.Browser.DataDir)
	ti2.Width = 40
	m.inputs["browserDataDir"] = ti2
}

func (m *configPanelModel) focusEditField() {
	for i, field := range m.editFields {
		if input, ok := m.inputs[field]; ok {
			if i == m.editField {
				input.Focus()
			} else {
				input.Blur()
			}
			m.inputs[field] = input
		}
	}
}

func (m *configPanelModel) saveEdit() {
	switch m.category {
	case 0:
		if prov, ok := config.DefaultCfg.Providers[m.editKey]; ok {
			prov.Type = m.selectVal["type"]
			prov.APIKey = m.inputs["apiKey"].Value()
			prov.BaseURL = m.inputs["baseUrl"].Value()
		}
	case 1:
		if agent, ok := config.DefaultCfg.Agents[m.editKey]; ok {
			agent.Provider = m.selectVal["provider"]
			agent.Model = m.inputs["model"].Value()
			agent.Workspace = m.inputs["workspace"].Value()
		}
	case 2:
		if ch, ok := config.DefaultCfg.Channels[m.editKey]; ok {
			ch.Type = m.selectVal["type"]
			ch.Agent = m.selectVal["agent"]
			ch.Enabled = m.boolVal["enabled"]
			if input, ok := m.inputs["appID"]; ok {
				ch.AppID = input.Value()
			}
			if input, ok := m.inputs["appSecret"]; ok {
				ch.AppSecret = input.Value()
			}
			if input, ok := m.inputs["hostAddress"]; ok {
				ch.HostAddress = input.Value()
			}
			if input, ok := m.inputs["token"]; ok {
				ch.Token = input.Value()
			}
		}
	case 3:
		config.DefaultCfg.Skill.GlobalPath = m.inputs["skillsPath"].Value()
		config.DefaultCfg.Browser.DataDir = m.inputs["browserDataDir"].Value()
	}
}

func (m configPanelModel) View() string {
	if m.showSaveConfirm {
		return m.renderSaveConfirm()
	}

	sidebarWidth := m.width/4 - 2
	contentWidth := m.width*3/4 - 2
	contentHeight := m.height - 6

	sidebar := m.renderSidebar(sidebarWidth, contentHeight)
	content := m.renderContent(contentWidth, contentHeight)
	help := m.renderHelp()

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(sidebarWidth).Render(sidebar),
		lipgloss.NewStyle().Width(contentWidth).Render(content),
	)

	return fmt.Sprintf("%s\n%s", mainContent, help)
}

func (m configPanelModel) renderSaveConfirm() string {
	confirmBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#ffb86c")).
		Padding(1, 2).
		Width(60)

	var sb strings.Builder
	sb.WriteString("⚠️  配置保存确认\n\n")
	if m.configChanged {
		sb.WriteString("检测到配置已修改。\n")
		sb.WriteString("保存后需要重启程序才能生效。\n\n")
	} else {
		sb.WriteString("配置未修改。\n\n")
	}
	sb.WriteString("是否保存并退出？ [Y/n]")

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(confirmBoxStyle.Render(sb.String()))
}

func (m configPanelModel) renderSidebar(width, height int) string {
	var sb strings.Builder

	categories := []string{"📡 Provider", "🤖 Agent", "💬 Channel", "⚙️ System"}
	for i, cat := range categories {
		style := categoryStyle
		if i == m.category && m.focus == focusSidebar {
			style = categoryActiveStyle
		}
		sb.WriteString(style.Render(cat))
		sb.WriteString("\n")
	}

	content := sb.String()

	if m.focus == focusSidebar {
		return sidebarActiveStyle.
			Width(width - 2).
			Height(height - 2).
			Render(content)
	}
	return sidebarStyle.
		Width(width - 2).
		Height(height - 2).
		Render(content)
}

func (m configPanelModel) renderContent(width, height int) string {
	var content string
	if m.editMode {
		content = m.renderEditForm(width, height)
	} else {
		content = m.renderBrowseView(width, height)
	}

	if m.focus == focusContent {
		return contentActiveStyle.
			Width(width - 2).
			Height(height - 2).
			Render(content)
	}
	return contentStyle.
		Width(width - 2).
		Height(height - 2).
		Render(content)
}

func (m configPanelModel) renderBrowseView(width, height int) string {
	var sb strings.Builder

	switch m.category {
	case 0:
		sb.WriteString(cfgLabelStyle.Render("📡 Providers"))
		sb.WriteString("\n\n")
		if len(m.items) == 0 {
			sb.WriteString(cfgHelpStyle.Render("暂无配置"))
		} else {
			for i, item := range m.items {
				style := cfgItemStyle
				if i == m.selectedIdx && m.focus == focusContent {
					style = cfgItemActiveStyle
				}
				prov := config.DefaultCfg.Providers[item]
				sb.WriteString(style.Render(fmt.Sprintf("• %s (%s)", item, prov.Type)))
				sb.WriteString("\n")
			}
		}

	case 1:
		sb.WriteString(cfgLabelStyle.Render("🤖 Agents"))
		sb.WriteString("\n\n")
		if len(m.items) == 0 {
			sb.WriteString(cfgHelpStyle.Render("暂无配置"))
		} else {
			for i, item := range m.items {
				style := cfgItemStyle
				if i == m.selectedIdx && m.focus == focusContent {
					style = cfgItemActiveStyle
				}
				agent := config.DefaultCfg.Agents[item]
				sb.WriteString(style.Render(fmt.Sprintf("• %s (%s/%s)", item, agent.Provider, agent.Model)))
				sb.WriteString("\n")
			}
		}

	case 2:
		sb.WriteString(cfgLabelStyle.Render("💬 Channels"))
		sb.WriteString("\n\n")
		if len(m.items) == 0 {
			sb.WriteString(cfgHelpStyle.Render("暂无配置"))
		} else {
			for i, item := range m.items {
				style := cfgItemStyle
				if i == m.selectedIdx && m.focus == focusContent {
					style = cfgItemActiveStyle
				}
				ch := config.DefaultCfg.Channels[item]
				status := "禁用"
				if ch.Enabled {
					status = "启用"
				}
				sb.WriteString(style.Render(fmt.Sprintf("• %s (%s) - %s", item, ch.Type, status)))
				sb.WriteString("\n")
			}
		}

	case 3:
		sb.WriteString(cfgLabelStyle.Render("⚙️ System"))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("Skills 路径: %s\n", config.DefaultCfg.Skill.GlobalPath))
		sb.WriteString(fmt.Sprintf("Browser 数据目录: %s\n", config.DefaultCfg.Browser.DataDir))
	}

	return sb.String()
}

func (m configPanelModel) renderEditForm(width, height int) string {
	var sb strings.Builder

	sb.WriteString(cfgLabelStyle.Render(fmt.Sprintf("编辑: %s", m.editKey)))
	sb.WriteString("\n\n")

	for i, field := range m.editFields {
		style := cfgItemStyle
		if i == m.editField {
			style = cfgItemActiveStyle
		}

		label := m.getFieldLabel(field)
		sb.WriteString(style.Render(label + ": "))
		sb.WriteString(" ")

		switch m.editFieldType[field] {
		case "select":
			sb.WriteString(cfgInputStyle.Render(m.selectVal[field]))
			sb.WriteString(" ")
			sb.WriteString(cfgHelpStyle.Render("[←→切换]"))
		case "bool":
			val := "false"
			if m.boolVal[field] {
				val = "true"
			}
			sb.WriteString(cfgInputStyle.Render(val))
			sb.WriteString(" ")
			sb.WriteString(cfgHelpStyle.Render("[Space/←→切换]"))
		default:
			if input, ok := m.inputs[field]; ok {
				sb.WriteString(cfgInputStyle.Render(input.View()))
			}
		}
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func (m configPanelModel) getFieldLabel(field string) string {
	labels := map[string]string{
		"type":           "类型",
		"apiKey":         "API Key",
		"baseUrl":        "Base URL",
		"provider":       "Provider",
		"model":          "模型",
		"workspace":      "工作目录",
		"agent":          "Agent",
		"enabled":        "启用",
		"appID":          "App ID",
		"appSecret":      "App Secret",
		"hostAddress":    "监听地址",
		"token":          "Token",
		"skillsPath":     "Skills 路径",
		"browserDataDir": "Browser 数据目录",
	}
	if label, ok := labels[field]; ok {
		return label
	}
	return field
}

func (m configPanelModel) renderHelp() string {
	if m.editMode {
		return cfgHelpStyle.Render("[↑↓切换字段] [←→切换选项] [Space切换布尔] [Enter保存] [Esc取消]")
	}
	if m.focus == focusSidebar {
		return cfgHelpStyle.Render("[↑↓选择分类] [Enter进入] [F10保存退出] [Tab切换视图]")
	}
	return cfgHelpStyle.Render("[↑↓选择项目] [Enter编辑] [Esc返回] [F10保存退出] [Tab切换视图]")
}
