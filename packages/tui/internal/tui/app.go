package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/cluesmith-ai/codev-tui/internal/api"
	"github.com/cluesmith-ai/codev-tui/internal/zellij"

	tea "github.com/charmbracelet/bubbletea"
)

type viewID int

const (
	viewBuilders viewID = iota
	viewTerminals
	viewAttention
	viewBacklog
	viewClosed
)

var viewNames = []string{"Builders", "Terminals", "Attention", "Backlog", "Closed"}

// Bubbletea messages

type overviewMsg struct{ data *api.OverviewData }
type stateMsg struct{ data *api.DashboardState }
type workspacesMsg struct{ data []api.WorkspaceInfo }
type sseEventMsg struct{ event api.SSEEvent }
type errMsg struct{ err error }
type sendResultMsg struct {
	ok  bool
	msg string
}
type openAllTabsMsg struct{ err error }
type newShellMsg struct {
	terminalID string
	err        error
}
type refreshDoneMsg struct{ err error }

// App is the root Bubbletea model.
type App struct {
	client    *api.Client
	sseCancel context.CancelFunc

	workspace string
	towerURL  string
	overview  *api.OverviewData
	state     *api.DashboardState
	connected bool
	width     int
	height    int

	activeView viewID
	builders   BuildersView
	terminals  TerminalsView
	attention  AttentionView
	backlog    BacklogView
	closed     ClosedView
	statusBar  StatusBar
	wsModal    WorkspaceModal
	sendModal  SendModal
}

func NewApp(client *api.Client, workspace string) App {
	return App{
		client:    client,
		workspace: workspace,
		towerURL:  client.BaseURL,
		builders:  NewBuildersView(),
		terminals: NewTerminalsView(),
		attention: NewAttentionView(),
		backlog:   NewBacklogView(),
		closed:    NewClosedView(),
		wsModal:   NewWorkspaceModal(),
		sendModal: NewSendModal(),
		statusBar: StatusBar{WorkspacePath: workspace},
	}
}

func (m App) Init() tea.Cmd {
	cmds := []tea.Cmd{startSSE(m.client)}

	if m.workspace == "" {
		cmds = append(cmds, fetchWorkspaces(m.client))
	} else {
		cmds = append(cmds,
			fetchOverview(m.client, m.workspace),
			fetchState(m.client, m.workspace),
		)
	}

	return tea.Batch(cmds...)
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.builders.SetHeight(msg.Height - 4)
		return m, nil

	case overviewMsg:
		m.overview = msg.data
		m.connected = true
		m.statusBar.Connected = true
		if m.overview != nil {
			m.builders.SetBuilders(m.overview.Builders)
			m.attention.SetData(m.overview)
			m.backlog.SetItems(m.overview.Backlog)
			m.closed.SetItems(m.overview.RecentlyClosed)
		}
		return m, nil

	case stateMsg:
		m.state = msg.data
		m.connected = true
		m.statusBar.Connected = true
		m.terminals.SetState(m.state)
		return m, nil

	case workspacesMsg:
		m.wsModal.SetWorkspaces(msg.data)
		m.wsModal.Show()
		return m, nil

	case sseEventMsg:
		m.connected = true
		m.statusBar.Connected = true
		return m, tea.Batch(
			fetchOverview(m.client, m.workspace),
			fetchState(m.client, m.workspace),
			listenSSE(m.client),
		)

	case errMsg:
		m.statusBar.Connected = false
		m.connected = false
		return m, nil

	case openAllTabsMsg:
		return m, nil

	case newShellMsg:
		if msg.err == nil && msg.terminalID != "" {
			zellij.AttachToTerminal(m.towerURL, msg.terminalID, m.workspace)
		}
		return m, fetchState(m.client, m.workspace)

	case refreshDoneMsg:
		return m, tea.Batch(
			fetchOverview(m.client, m.workspace),
			fetchState(m.client, m.workspace),
		)

	case sendResultMsg:
		if msg.ok {
			m.sendModal.SetFeedback(msg.msg)
		} else {
			m.sendModal.SetError(msg.msg)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.wsModal.IsVisible() {
		return m.handleWorkspaceKey(msg)
	}
	if m.sendModal.IsVisible() {
		return m.handleSendKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab":
		m.activeView = (m.activeView + 1) % viewID(len(viewNames))
		return m, nil

	case "shift+tab":
		m.activeView = (m.activeView - 1 + viewID(len(viewNames))) % viewID(len(viewNames))
		return m, nil

	case "w":
		return m, fetchWorkspacesAndShow(m.client)

	case "s":
		m.sendModal.Show()
		return m, nil

	case "r":
		return m, tea.Batch(
			fetchOverview(m.client, m.workspace),
			fetchState(m.client, m.workspace),
		)

	case "R":
		return m, forceRefresh(m.client, m.workspace)

	case "n":
		if m.workspace != "" {
			return m, createNewShell(m.client, m.workspace)
		}
		return m, nil

	case "a":
		m.openArchitectTerminal()
		return m, nil

	case "W":
		return m, openAllWorkspaceTabs(m.client, m.towerURL)

	case "up", "k":
		m.moveUp()
		return m, nil

	case "down", "j":
		m.moveDown()
		return m, nil

	case "enter":
		return m.handleEnter()
	}

	return m, nil
}

func (m *App) openArchitectTerminal() {
	if m.state == nil {
		return
	}
	for _, arch := range m.state.Architects {
		if arch.TerminalID != "" {
			zellij.AttachToTerminal(m.towerURL, arch.TerminalID, m.workspace)
			return
		}
	}
}

func (m *App) moveUp() {
	switch m.activeView {
	case viewBuilders:
		m.builders.Up()
	case viewTerminals:
		m.terminals.Up()
	case viewAttention:
		m.attention.Up()
	case viewBacklog:
		m.backlog.Up()
	case viewClosed:
		m.closed.Up()
	}
}

func (m *App) moveDown() {
	switch m.activeView {
	case viewBuilders:
		m.builders.Down()
	case viewTerminals:
		m.terminals.Down()
	case viewAttention:
		m.attention.Down()
	case viewBacklog:
		m.backlog.Down()
	case viewClosed:
		m.closed.Down()
	}
}

func (m App) handleEnter() (tea.Model, tea.Cmd) {
	switch m.activeView {
	case viewBuilders:
		if b := m.builders.Selected(); b != nil {
			zellij.AttachToTerminal(m.towerURL, findBuilderTerminalID(m.state, b.ID), m.workspace)
		}
	case viewTerminals:
		if t := m.terminals.Selected(); t != nil {
			zellij.AttachToTerminal(m.towerURL, t.TerminalID, m.workspace)
		}
	}
	return m, nil
}

func findBuilderTerminalID(state *api.DashboardState, builderID string) string {
	if state == nil {
		return ""
	}
	for _, b := range state.Builders {
		if b.ID == builderID || b.Name == builderID {
			return b.TerminalID
		}
	}
	return ""
}

func (m App) handleWorkspaceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.wsModal.Hide()
	case "enter":
		if ws := m.wsModal.Selected(); ws != nil {
			m.workspace = ws.Path
			m.statusBar.WorkspacePath = ws.Path
			m.statusBar.WorkspaceName = ws.Name
			m.wsModal.Hide()
			return m, tea.Batch(
				fetchOverview(m.client, m.workspace),
				fetchState(m.client, m.workspace),
			)
		}
	case "up":
		m.wsModal.Up()
	case "down":
		m.wsModal.Down()
	case "backspace":
		m.wsModal.DeleteFilterChar()
	default:
		if len(msg.String()) == 1 {
			m.wsModal.AddFilterChar(rune(msg.String()[0]))
		}
	}
	return m, nil
}

func (m App) handleSendKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.sendModal.Hide()
	case "tab":
		m.sendModal.SwitchFocus()
	case "enter":
		target := m.sendModal.Target()
		message := m.sendModal.Message()
		if target != "" && message != "" {
			return m, sendMessage(m.client, target, message, m.workspace)
		}
	case "backspace":
		m.sendModal.DeleteChar()
	default:
		if len(msg.String()) == 1 {
			m.sendModal.AddChar(rune(msg.String()[0]))
		}
	}
	return m, nil
}

func (m App) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.wsModal.IsVisible() {
		return m.wsModal.View(m.width, m.height)
	}
	if m.sendModal.IsVisible() {
		return m.sendModal.View(m.width, m.height)
	}

	var b strings.Builder

	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	switch m.activeView {
	case viewBuilders:
		b.WriteString(m.builders.View(m.width))
	case viewTerminals:
		b.WriteString(m.terminals.View(m.width))
	case viewAttention:
		b.WriteString(m.attention.View(m.width))
	case viewBacklog:
		b.WriteString(m.backlog.View(m.width))
	case viewClosed:
		b.WriteString(m.closed.View(m.width))
	}

	lines := strings.Count(b.String(), "\n")
	for i := lines; i < m.height-2; i++ {
		b.WriteString("\n")
	}

	b.WriteString(m.statusBar.View(m.width))

	return b.String()
}

func (m App) renderTabs() string {
	var tabs []string
	for i, name := range viewNames {
		if viewID(i) == m.activeView {
			tabs = append(tabs, StyleTabActive.Render(name))
		} else {
			tabs = append(tabs, StyleTabInactive.Render(name))
		}
	}
	return "  " + strings.Join(tabs, " ")
}

// --- Commands ---

func fetchOverview(client *api.Client, workspace string) tea.Cmd {
	return func() tea.Msg {
		data, err := client.Overview(workspace)
		if err != nil {
			return errMsg{err: err}
		}
		return overviewMsg{data: data}
	}
}

func fetchState(client *api.Client, workspace string) tea.Cmd {
	return func() tea.Msg {
		if workspace == "" {
			return nil
		}
		data, err := client.WorkspaceState(workspace)
		if err != nil {
			return errMsg{err: err}
		}
		return stateMsg{data: data}
	}
}

func fetchWorkspaces(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := client.Workspaces()
		if err != nil {
			return errMsg{err: err}
		}
		return workspacesMsg{data: data}
	}
}

func fetchWorkspacesAndShow(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := client.Workspaces()
		if err != nil {
			return errMsg{err: err}
		}
		return workspacesMsg{data: data}
	}
}

func startSSE(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		events := make(chan api.SSEEvent, 16)
		ctx := context.Background()
		client.StreamSSE(ctx, events)
		event := <-events
		return sseEventMsg{event: event}
	}
}

func listenSSE(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		events := make(chan api.SSEEvent, 16)
		ctx := context.Background()
		client.StreamSSE(ctx, events)
		event := <-events
		return sseEventMsg{event: event}
	}
}

func createNewShell(client *api.Client, workspace string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.CreateShellTab(workspace)
		if err != nil {
			return newShellMsg{err: err}
		}
		return newShellMsg{terminalID: resp.TerminalID}
	}
}

func forceRefresh(client *api.Client, workspace string) tea.Cmd {
	return func() tea.Msg {
		err := client.RefreshOverview(workspace)
		return refreshDoneMsg{err: err}
	}
}

func openAllWorkspaceTabs(client *api.Client, towerURL string) tea.Cmd {
	return func() tea.Msg {
		workspaces, err := client.Workspaces()
		if err != nil {
			return openAllTabsMsg{err: err}
		}

		var tabs []zellij.WorkspaceTabInfo
		for _, ws := range workspaces {
			if !ws.Active {
				continue
			}
			archTermID := ""
			state, err := client.WorkspaceState(ws.Path)
			if err == nil && state != nil {
				for _, arch := range state.Architects {
					if arch.TerminalID != "" {
						archTermID = arch.TerminalID
						break
					}
				}
			}
			tabs = append(tabs, zellij.WorkspaceTabInfo{
				Name:                ws.Name,
				Path:                ws.Path,
				ArchitectTerminalID: archTermID,
			})
		}

		if len(tabs) == 0 {
			return openAllTabsMsg{err: fmt.Errorf("no active workspaces")}
		}

		err = zellij.OpenAllWorkspaceTabs(towerURL, tabs)
		return openAllTabsMsg{err: err}
	}
}

func sendMessage(client *api.Client, to, message, workspace string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.Send(api.SendRequest{
			To:        to,
			Message:   message,
			Workspace: workspace,
		})
		if err != nil {
			return sendResultMsg{ok: false, msg: err.Error()}
		}
		if !resp.OK {
			return sendResultMsg{ok: false, msg: resp.Message}
		}
		return sendResultMsg{ok: true, msg: fmt.Sprintf("Sent to %s", resp.ResolvedTo)}
	}
}
