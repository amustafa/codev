package tui

import (
	"fmt"
	"strings"

	"github.com/cluesmith-ai/codev-tui/internal/api"
)

type WorkspaceModal struct {
	workspaces []api.WorkspaceInfo
	cursor     int
	visible    bool
	filter     string
	filtered   []api.WorkspaceInfo
}

func NewWorkspaceModal() WorkspaceModal {
	return WorkspaceModal{}
}

func (m *WorkspaceModal) SetWorkspaces(ws []api.WorkspaceInfo) {
	m.workspaces = ws
	m.applyFilter()
}

func (m *WorkspaceModal) Show() {
	m.visible = true
	m.filter = ""
	m.cursor = 0
	m.applyFilter()
}

func (m *WorkspaceModal) Hide() {
	m.visible = false
}

func (m *WorkspaceModal) IsVisible() bool {
	return m.visible
}

func (m *WorkspaceModal) Up() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *WorkspaceModal) Down() {
	if m.cursor < len(m.filtered)-1 {
		m.cursor++
	}
}

func (m *WorkspaceModal) AddFilterChar(ch rune) {
	m.filter += string(ch)
	m.cursor = 0
	m.applyFilter()
}

func (m *WorkspaceModal) DeleteFilterChar() {
	if len(m.filter) > 0 {
		m.filter = m.filter[:len(m.filter)-1]
		m.cursor = 0
		m.applyFilter()
	}
}

func (m *WorkspaceModal) Selected() *api.WorkspaceInfo {
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		return &m.filtered[m.cursor]
	}
	return nil
}

func (m *WorkspaceModal) applyFilter() {
	if m.filter == "" {
		m.filtered = m.workspaces
		return
	}
	lower := strings.ToLower(m.filter)
	m.filtered = nil
	for _, ws := range m.workspaces {
		if strings.Contains(strings.ToLower(ws.Name), lower) ||
			strings.Contains(strings.ToLower(ws.Path), lower) {
			m.filtered = append(m.filtered, ws)
		}
	}
}

func (m WorkspaceModal) View(width, height int) string {
	if !m.visible {
		return ""
	}

	var b strings.Builder
	b.WriteString(StyleModalTitle.Render("Select Workspace"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Filter: %s█\n\n", m.filter))

	if len(m.filtered) == 0 {
		b.WriteString(StyleNormal.Render("  No matching workspaces"))
	} else {
		for i, ws := range m.filtered {
			selected := i == m.cursor
			prefix := "  "
			if selected {
				prefix = "▶ "
			}

			status := StyleStatusDisconnected.Render("○")
			if ws.Active {
				status = StyleStatusConnected.Render("●")
			}

			line := fmt.Sprintf("%s%s %-20s %s", prefix, status, ws.Name, StyleNormal.Render(ws.Path))

			if selected {
				b.WriteString(StyleSelected.Render(line))
			} else {
				b.WriteString(StyleNormal.Render(line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(StyleHelp.Render("  [enter] select  [esc] cancel  [type] filter"))

	content := b.String()
	modal := StyleModal.Render(content)

	padTop := max(0, (height-strings.Count(modal, "\n"))/2)
	padLeft := max(0, (width-60)/2)

	var out strings.Builder
	for i := 0; i < padTop; i++ {
		out.WriteString("\n")
	}
	for _, line := range strings.Split(modal, "\n") {
		out.WriteString(strings.Repeat(" ", padLeft))
		out.WriteString(line)
		out.WriteString("\n")
	}

	return out.String()
}
