package tui

import (
	"fmt"
	"runtime"
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
	monitorView
	configView
)

type runtimeModel struct {
	view         runtimeView
	width        int
	height       int
	chatInput    textinput.Model
	chatViewport viewport.Model
	logViewport  viewport.Model
	chatMessages []chatMessage
	configModel  model
	tuiChannel   *TUIChannel
	agentName    string
	isProcessing bool
	startTime    time.Time
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

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6272a4")).
			Padding(0, 1)

	infoLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B5CF6")).
			Bold(true)

	infoValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fafafa"))

	statusOkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50fa7b"))

	statusWarnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffb86c"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272a4")).
			Background(lipgloss.Color("#1a1b26")).
			Padding(0, 1)
)

type agentResponseMsg struct {
	content string
}

type tickMsg struct{}

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
		startTime:    time.Now(),
	}
}

func (m runtimeModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		waitForAgentResponse(m.tuiChannel),
		tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg{}
		}),
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
		m.logViewport.Width = (m.width * 2 / 3) - 6
		m.logViewport.Height = m.height - 10
		m.chatInput.Width = m.width - 6
		m.configModel.width = m.width
		m.configModel.height = m.height

	case tickMsg:
		if m.view == monitorView {
			m.updateLogViewport()
		}
		cmds = append(cmds, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg{}
		}))

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
			m.view = (m.view + 1) % 3
			if m.view == monitorView {
				m.updateLogViewport()
			}
			return m, nil
		case "shift+tab":
			m.view = (m.view + 2) % 3
			if m.view == monitorView {
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
	tabNames := []string{"💬 聊天", "📊 监控", "⚙️ 配置"}
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
	case monitorView:
		content = m.renderMonitorView()
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

func (m runtimeModel) renderMonitorView() string {
	leftWidth := m.width/3 - 2
	rightWidth := m.width*2/3 - 2
	contentHeight := m.height - 6

	leftPanel := m.renderInfoPanel(leftWidth, contentHeight)
	rightPanel := m.renderLogPanel(rightWidth, contentHeight)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftWidth).Render(leftPanel),
		lipgloss.NewStyle().Width(rightWidth).Render(rightPanel),
	)

	statusBar := m.renderStatusBar()

	return fmt.Sprintf("%s\n%s", mainContent, statusBar)
}

func (m runtimeModel) renderInfoPanel(width, height int) string {
	var sb strings.Builder

	sb.WriteString(infoLabelStyle.Render("🤖 Agent"))
	sb.WriteString("\n")
	sb.WriteString(infoValueStyle.Render("  "))
	sb.WriteString(infoValueStyle.Render(m.agentName))
	sb.WriteString("\n\n")

	uptime := time.Since(m.startTime)
	sb.WriteString(infoLabelStyle.Render("⏱️ 运行时长"))
	sb.WriteString("\n")
	sb.WriteString(infoValueStyle.Render("  "))
	sb.WriteString(infoValueStyle.Render(formatDuration(uptime)))
	sb.WriteString("\n\n")

	sb.WriteString(infoLabelStyle.Render("📡 Providers"))
	sb.WriteString("\n")
	for name, prov := range config.DefaultCfg.Providers {
		sb.WriteString(infoValueStyle.Render(fmt.Sprintf("  • %s (%s)\n", name, prov.Type)))
	}
	sb.WriteString("\n")

	sb.WriteString(infoLabelStyle.Render("💬 Channels"))
	sb.WriteString("\n")
	for name, ch := range config.DefaultCfg.Channels {
		status := statusWarnStyle.Render("禁用")
		if ch.Enabled {
			status = statusOkStyle.Render("启用")
		}
		sb.WriteString(infoValueStyle.Render(fmt.Sprintf("  • %s ", name)))
		sb.WriteString(status)
		sb.WriteString("\n")
	}

	content := sb.String()

	return panelStyle.
		Width(width - 2).
		Height(height - 2).
		Render(content)
}

func (m runtimeModel) renderLogPanel(width, height int) string {
	return panelStyle.
		Width(width - 2).
		Height(height - 2).
		Render(m.logViewport.View())
}

func (m runtimeModel) renderStatusBar() string {
	var mbs runtime.MemStats
	runtime.ReadMemStats(&mbs)

	allocMB := float64(mbs.Alloc) / 1024 / 1024
	sysMB := float64(mbs.Sys) / 1024 / 1024
	numGoroutine := runtime.NumGoroutine()

	memInfo := fmt.Sprintf("内存: %.1fMB / %.1fMB | Goroutines: %d", allocMB, sysMB, numGoroutine)
	uptimeInfo := fmt.Sprintf("运行: %s", formatDuration(time.Since(m.startTime)))

	leftPart := statusBarStyle.Render(memInfo)
	rightPart := statusBarStyle.Render(uptimeInfo)

	width := m.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart) - 2
	if width < 0 {
		width = 0
	}
	middlePart := lipgloss.NewStyle().Width(width).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Bottom, leftPart, middlePart, rightPart)
}

func (m *runtimeModel) updateLogViewport() {
	logs := logger.GetRecentLogs(50000)
	m.logViewport.SetContent(logs)
	m.logViewport.GotoBottom()
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%d时%d分%d秒", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%d分%d秒", m, s)
	}
	return fmt.Sprintf("%d秒", s)
}
