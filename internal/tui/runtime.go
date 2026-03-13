package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yockii/wangshu/internal/config"
	"github.com/yockii/wangshu/pkg/logger"
)

type runtimeView int

const (
	chatView runtimeView = iota
	statusView
	logView
	configView
)

type runtimeModel struct {
	view          runtimeView
	width         int
	height        int
	chatInput     textinput.Model
	chatViewport  viewport.Model
	logViewport   viewport.Model
	chatMessages  []chatMessage
	statusContent string
	configModel   model
	tuiChannel    *TUIChannel
	agentName     string
	isProcessing  bool
}

type chatMessage struct {
	sender  string
	content string
	time    time.Time
	isUser  bool
}

var (
	chatBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8B5CF6")).
			Padding(0, 1)

	userMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8be9fd")).
			Bold(true)

	agentMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50fa7b")).
			Bold(true)

	msgTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272a4")).
			Faint(true)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#ff79c6")).
			Padding(0, 1)

	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272a4")).
			Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fafafa")).
			Background(lipgloss.Color("#8B5CF6")).
			Padding(0, 2).
			Bold(true)

	processingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffb86c")).
			Faint(true)
)

type agentResponseMsg struct {
	content string
}

func newRuntimeModel(tuiChannel *TUIChannel, agentName string) runtimeModel {
	ti := textinput.New()
	ti.Placeholder = "输入消息..."
	ti.Focus()
	ti.Width = 80

	vp := viewport.New(80, 20)
	logVp := viewport.New(80, 20)

	cfgModel := initialModel()
	cfgModel.state = mainMenuState

	return runtimeModel{
		view:         chatView,
		chatInput:    ti,
		chatViewport: vp,
		logViewport:  logVp,
		chatMessages: make([]chatMessage, 0),
		tuiChannel:   tuiChannel,
		agentName:    agentName,
		configModel:  cfgModel,
	}
}

func (m runtimeModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		waitForAgentResponse(m.tuiChannel),
	)
}

func waitForAgentResponse(ch *TUIChannel) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch.ReceiveOutbound()
		if !ok {
			return nil
		}
		return agentResponseMsg{content: msg.Content}
	}
}

func (m runtimeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.chatViewport.Width = m.width - 4
		m.chatViewport.Height = m.height - 8
		m.logViewport.Width = m.width - 4
		m.logViewport.Height = m.height - 6
		m.chatInput.Width = m.width - 6
		m.configModel.width = m.width
		m.configModel.height = m.height

	case agentResponseMsg:
		if msg.content != "" {
			m.chatMessages = append(m.chatMessages, chatMessage{
				sender:  m.agentName,
				content: msg.content,
				time:    time.Now(),
				isUser:  false,
			})
			m.isProcessing = false
			m.updateChatViewport()
		}
		cmds = append(cmds, waitForAgentResponse(m.tuiChannel))

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.view = (m.view + 1) % 4
			if m.view == logView {
				m.updateLogViewport()
			}
			return m, nil
		case "shift+tab":
			m.view = (m.view + 3) % 4
			if m.view == logView {
				m.updateLogViewport()
			}
			return m, nil
		case "ctrl+l":
			if m.view == chatView {
				m.chatMessages = make([]chatMessage, 0)
				m.updateChatViewport()
			}
			return m, nil
		}

		if m.view == chatView {
			return m.updateChatView(msg)
		} else if m.view == configView {
			var cmd tea.Cmd
			newModel, cmd := m.configModel.Update(msg)
			if newModel, ok := newModel.(model); ok {
				m.configModel = newModel
			}
			if m.configModel.state == doneState {
				return m, tea.Quit
			}
			return m, cmd
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *runtimeModel) updateChatView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "enter":
		content := strings.TrimSpace(m.chatInput.Value())
		if content != "" && !m.isProcessing {
			m.chatMessages = append(m.chatMessages, chatMessage{
				sender:  "You",
				content: content,
				time:    time.Now(),
				isUser:  true,
			})
			m.chatInput.SetValue("")
			m.updateChatViewport()
			m.isProcessing = true

			m.tuiChannel.PublishUserMessage(content)
		}
	default:
		m.chatInput, cmd = m.chatInput.Update(msg)
	}

	return m, cmd
}

func (m *runtimeModel) updateChatViewport() {
	var sb strings.Builder
	for _, msg := range m.chatMessages {
		timeStr := msg.time.Format("15:04:05")
		var senderStyle lipgloss.Style
		if msg.isUser {
			senderStyle = userMsgStyle
		} else {
			senderStyle = agentMsgStyle
		}

		sb.WriteString(msgTimeStyle.Render(timeStr))
		sb.WriteString(" ")
		sb.WriteString(senderStyle.Render(msg.sender))
		sb.WriteString("\n")
		sb.WriteString(msg.content)
		sb.WriteString("\n\n")
	}
	m.chatViewport.SetContent(sb.String())
	m.chatViewport.GotoBottom()
}

func (m runtimeModel) View() string {
	var tabs []string
	tabNames := []string{"💬 聊天", "📊 状态", "📋 日志", "⚙️ 配置"}
	for i, name := range tabNames {
		if i == int(m.view) {
			tabs = append(tabs, activeTabStyle.Render(name))
		} else {
			tabs = append(tabs, tabStyle.Render(name))
		}
	}
	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	var content string
	switch m.view {
	case chatView:
		content = m.renderChatView()
	case statusView:
		content = m.renderStatusView()
	case logView:
		content = m.renderLogView()
	case configView:
		content = m.configModel.View()
	}

	return fmt.Sprintf("%s\n%s", tabRow, content)
}

func (m runtimeModel) renderChatView() string {
	chatBox := chatBoxStyle.
		Width(m.width - 2).
		Height(m.height - 6).
		Render(m.chatViewport.View())

	inputPrompt := "> "
	if m.isProcessing {
		inputPrompt = processingStyle.Render("⏳ Agent思考中... ") + "> "
	}

	inputBox := inputBoxStyle.
		Width(m.width - 2).
		Render(inputPrompt + m.chatInput.View())

	helpText := helpStyle.Render("[Enter发送] [Tab切换视图] [Ctrl+L清屏] [Ctrl+Q退出]")

	return fmt.Sprintf("%s\n%s\n%s", chatBox, inputBox, helpText)
}

func (m runtimeModel) renderStatusView() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("📊 系统状态"))
	sb.WriteString("\n\n")

	sb.WriteString(labelStyle.Render("🤖 Agent: "))
	sb.WriteString(m.agentName)
	sb.WriteString("\n\n")

	sb.WriteString(labelStyle.Render("📡 Providers: "))
	sb.WriteString(fmt.Sprintf("%d 个配置", len(config.DefaultCfg.Providers)))
	sb.WriteString("\n")
	for name, prov := range config.DefaultCfg.Providers {
		sb.WriteString(fmt.Sprintf("  • %s (%s)\n", name, prov.Type))
	}

	sb.WriteString("\n")
	sb.WriteString(labelStyle.Render("💬 Channels: "))
	sb.WriteString(fmt.Sprintf("%d 个配置", len(config.DefaultCfg.Channels)))
	sb.WriteString("\n")
	for name, ch := range config.DefaultCfg.Channels {
		status := "禁用"
		if ch.Enabled {
			status = "启用"
		}
		sb.WriteString(fmt.Sprintf("  • %s (%s) - %s\n", name, ch.Type, status))
	}

	sb.WriteString("\n\n")
	sb.WriteString(helpStyle.Render("[Tab切换视图] [Ctrl+Q退出]"))

	return docStyle.Render(sb.String())
}

func (m *runtimeModel) updateLogViewport() {
	logs := logger.GetRecentLogs(50000)
	m.logViewport.SetContent(logs)
	m.logViewport.GotoBottom()
}

func (m runtimeModel) renderLogView() string {
	logBox := chatBoxStyle.
		Width(m.width - 2).
		Height(m.height - 4).
		Render(m.logViewport.View())

	helpText := helpStyle.Render("[Tab切换视图] [↑↓滚动] [Ctrl+Q退出]")

	return fmt.Sprintf("%s\n%s", logBox, helpText)
}
