package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/cluesmith-ai/codev-tui/internal/api"
)

type BuildersView struct {
	builders []api.OverviewBuilder
	cursor   int
	height   int
	offset   int
}

func NewBuildersView() BuildersView {
	return BuildersView{}
}

func (v *BuildersView) SetBuilders(builders []api.OverviewBuilder) {
	v.builders = builders
	if v.cursor >= len(v.builders) {
		v.cursor = max(0, len(v.builders)-1)
	}
}

func (v *BuildersView) SetHeight(h int) {
	v.height = h
}

func (v *BuildersView) Up() {
	if v.cursor > 0 {
		v.cursor--
		if v.cursor < v.offset {
			v.offset = v.cursor
		}
	}
}

func (v *BuildersView) Down() {
	if v.cursor < len(v.builders)-1 {
		v.cursor++
		visible := v.visibleRows()
		if v.cursor >= v.offset+visible {
			v.offset = v.cursor - visible + 1
		}
	}
}

func (v *BuildersView) Selected() *api.OverviewBuilder {
	if v.cursor >= 0 && v.cursor < len(v.builders) {
		return &v.builders[v.cursor]
	}
	return nil
}

func (v *BuildersView) visibleRows() int {
	rows := v.height - 3
	if rows < 1 {
		rows = 10
	}
	return rows
}

func (v BuildersView) View(width int) string {
	if len(v.builders) == 0 {
		return StyleNormal.Render("  No active builders")
	}

	var b strings.Builder

	header := fmt.Sprintf("  %-6s %-25s %-12s %-10s %-5s %-12s %-8s %s",
		"ID", "Issue", "Phase", "Protocol", "Prog", "Blocked", "Idle", "Area")
	b.WriteString(StyleHeader.Render(header))
	b.WriteString("\n")
	b.WriteString(StyleDivider.Render(strings.Repeat("─", min(width, 100))))
	b.WriteString("\n")

	visible := v.visibleRows()
	end := min(v.offset+visible, len(v.builders))

	for i := v.offset; i < end; i++ {
		builder := v.builders[i]
		selected := i == v.cursor

		id := truncate(builder.ID, 6)
		issue := "—"
		if builder.IssueTitle != nil {
			issue = truncate(fmt.Sprintf("#%s %s", deref(builder.IssueID), *builder.IssueTitle), 25)
		}
		phase := truncate(builder.ProtocolPhase, 12)
		protocol := truncate(builder.Protocol, 10)
		progress := fmt.Sprintf("%d%%", builder.Progress)
		blocked := "—"
		if builder.Blocked != nil {
			blocked = truncate(*builder.Blocked, 12)
		}
		idle := formatDuration(builder.IdleMs)
		area := truncate(builder.Area, 15)

		line := fmt.Sprintf("  %-6s %-25s %-12s %-10s %-5s %-12s %-8s %s",
			id, issue, phase, protocol, progress, blocked, idle, area)

		if builder.PRReady {
			line += " " + StylePRReady.Render("[PR]")
		}

		if selected {
			line = "▶" + line[1:]
			b.WriteString(StyleSelected.Render(line))
		} else if builder.Blocked != nil {
			b.WriteString(StyleBlocked.Render(line))
		} else {
			b.WriteString(StyleActive.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatDuration(ms int64) string {
	if ms <= 0 {
		return "—"
	}
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
