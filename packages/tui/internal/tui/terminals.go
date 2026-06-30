package tui

import (
	"fmt"
	"strings"

	"github.com/cluesmith-ai/codev-tui/internal/api"
)

type terminalEntry struct {
	Kind       string // "architect", "builder", "shell"
	Label      string
	TerminalID string
	Name       string
}

type TerminalsView struct {
	entries []terminalEntry
	cursor  int
}

func NewTerminalsView() TerminalsView {
	return TerminalsView{}
}

func (v *TerminalsView) SetState(state *api.DashboardState) {
	v.entries = nil
	if state == nil {
		return
	}

	for _, arch := range state.Architects {
		if arch.TerminalID != "" {
			v.entries = append(v.entries, terminalEntry{
				Kind:       "architect",
				Label:      fmt.Sprintf("Architect (%s)", arch.Name),
				TerminalID: arch.TerminalID,
				Name:       arch.Name,
			})
		}
	}

	for _, b := range state.Builders {
		if b.TerminalID != "" {
			label := b.Name
			if label == "" {
				label = b.ID
			}
			v.entries = append(v.entries, terminalEntry{
				Kind:       "builder",
				Label:      fmt.Sprintf("Builder %s", label),
				TerminalID: b.TerminalID,
				Name:       b.Name,
			})
		}
	}

	for _, u := range state.Utils {
		if u.TerminalID != "" {
			v.entries = append(v.entries, terminalEntry{
				Kind:       "shell",
				Label:      u.Name,
				TerminalID: u.TerminalID,
				Name:       u.Name,
			})
		}
	}

	if v.cursor >= len(v.entries) {
		v.cursor = max(0, len(v.entries)-1)
	}
}

func (v *TerminalsView) Up() {
	if v.cursor > 0 {
		v.cursor--
	}
}

func (v *TerminalsView) Down() {
	if v.cursor < len(v.entries)-1 {
		v.cursor++
	}
}

func (v *TerminalsView) Selected() *terminalEntry {
	if v.cursor >= 0 && v.cursor < len(v.entries) {
		return &v.entries[v.cursor]
	}
	return nil
}

func (v TerminalsView) View(width int) string {
	if len(v.entries) == 0 {
		return StyleNormal.Render("  No active terminals")
	}

	var b strings.Builder

	header := fmt.Sprintf("  %-12s %-30s %-36s", "Type", "Name", "Terminal ID")
	b.WriteString(StyleHeader.Render(header))
	b.WriteString("\n")
	b.WriteString(StyleDivider.Render(strings.Repeat("─", min(width, 85))))
	b.WriteString("\n")

	for i, entry := range v.entries {
		selected := i == v.cursor
		prefix := "  "
		if selected {
			prefix = "▶ "
		}

		kindStyle := StyleNormal
		switch entry.Kind {
		case "architect":
			kindStyle = StylePRReady
		case "builder":
			kindStyle = StyleActive
		case "shell":
			kindStyle = StyleWarning
		}

		line := fmt.Sprintf("%s%-12s %-30s %-36s",
			prefix,
			kindStyle.Render(entry.Kind),
			truncate(entry.Label, 30),
			StyleNormal.Render(truncate(entry.TerminalID, 36)))

		if selected {
			b.WriteString(StyleSelected.Render(line))
		} else {
			b.WriteString(StyleNormal.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(StyleHelp.Render("  [enter] attach to terminal in new Zellij pane"))

	return b.String()
}
