package tui

import (
	"fmt"
	"strings"

	"github.com/cluesmith-ai/codev-tui/internal/api"
)

type ClosedView struct {
	items  []api.OverviewRecentlyClosed
	cursor int
}

func NewClosedView() ClosedView {
	return ClosedView{}
}

func (v *ClosedView) SetItems(items []api.OverviewRecentlyClosed) {
	v.items = items
	if v.cursor >= len(v.items) {
		v.cursor = max(0, len(v.items)-1)
	}
}

func (v *ClosedView) Up() {
	if v.cursor > 0 {
		v.cursor--
	}
}

func (v *ClosedView) Down() {
	if v.cursor < len(v.items)-1 {
		v.cursor++
	}
}

func (v *ClosedView) Selected() *api.OverviewRecentlyClosed {
	if v.cursor >= 0 && v.cursor < len(v.items) {
		return &v.items[v.cursor]
	}
	return nil
}

func (v ClosedView) View(width int) string {
	if len(v.items) == 0 {
		return StyleNormal.Render("  No recently closed items")
	}

	var b strings.Builder

	header := fmt.Sprintf("  %-6s %-45s %-12s %s", "ID", "Title", "Closed", "PR")
	b.WriteString(StyleHeader.Render(header))
	b.WriteString("\n")
	b.WriteString(StyleDivider.Render(strings.Repeat("─", min(width, 90))))
	b.WriteString("\n")

	for i, item := range v.items {
		selected := i == v.cursor
		prefix := "  "
		if selected {
			prefix = "▶ "
		}

		closed := truncate(item.ClosedAt, 10)
		pr := ""
		if item.PRUrl != "" {
			pr = StyleActive.Render("✓ PR")
		}

		line := fmt.Sprintf("%s#%-5s %-45s %-12s %s", prefix,
			item.ID,
			truncate(item.Title, 45),
			closed,
			pr)

		if selected {
			b.WriteString(StyleSelected.Render(line))
		} else {
			b.WriteString(StyleNormal.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}
