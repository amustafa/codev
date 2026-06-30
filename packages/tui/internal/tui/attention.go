package tui

import (
	"fmt"
	"strings"

	"github.com/cluesmith-ai/codev-tui/internal/api"
)

type AttentionView struct {
	prs      []api.OverviewPR
	blocked  []api.OverviewBuilder
	cursor   int
	total    int
	section  int // 0 = PRs, 1 = blocked
}

func NewAttentionView() AttentionView {
	return AttentionView{}
}

func (v *AttentionView) SetData(overview *api.OverviewData) {
	if overview == nil {
		v.prs = nil
		v.blocked = nil
		v.total = 0
		return
	}
	v.prs = overview.PendingPRs

	v.blocked = nil
	for _, b := range overview.Builders {
		if b.Blocked != nil {
			v.blocked = append(v.blocked, b)
		}
	}
	v.total = len(v.prs) + len(v.blocked)
	if v.cursor >= v.total {
		v.cursor = max(0, v.total-1)
	}
}

func (v *AttentionView) Up() {
	if v.cursor > 0 {
		v.cursor--
	}
}

func (v *AttentionView) Down() {
	if v.cursor < v.total-1 {
		v.cursor++
	}
}

func (v *AttentionView) SelectedPR() *api.OverviewPR {
	if v.cursor < len(v.prs) {
		return &v.prs[v.cursor]
	}
	return nil
}

func (v *AttentionView) SelectedBlockedBuilder() *api.OverviewBuilder {
	idx := v.cursor - len(v.prs)
	if idx >= 0 && idx < len(v.blocked) {
		return &v.blocked[idx]
	}
	return nil
}

func (v AttentionView) View(width int) string {
	if v.total == 0 {
		return StyleNormal.Render("  Nothing needs attention")
	}

	var b strings.Builder
	row := 0

	if len(v.prs) > 0 {
		b.WriteString(StyleHeader.Render("  PRs for Review"))
		b.WriteString("\n")
		for _, pr := range v.prs {
			selected := row == v.cursor
			prefix := "  "
			if selected {
				prefix = "▶ "
			}
			line := fmt.Sprintf("%s#%-5s %-40s %s", prefix,
				pr.ID,
				truncate(pr.Title, 40),
				StyleWarning.Render(pr.ReviewStatus))

			if pr.IsDraft {
				line += " " + StyleNormal.Render("[draft]")
			}

			if selected {
				b.WriteString(StyleSelected.Render(line))
			} else {
				b.WriteString(StyleNormal.Render(line))
			}
			b.WriteString("\n")
			row++
		}
		b.WriteString("\n")
	}

	if len(v.blocked) > 0 {
		b.WriteString(StyleHeader.Render("  Blocked at Gate"))
		b.WriteString("\n")
		for _, builder := range v.blocked {
			selected := row == v.cursor
			prefix := "  "
			if selected {
				prefix = "▶ "
			}
			gate := deref(builder.BlockedGate)
			since := ""
			if builder.BlockedSince != nil {
				since = " since " + truncate(*builder.BlockedSince, 16)
			}
			line := fmt.Sprintf("%s%-8s %-15s%s", prefix,
				truncate(builder.ID, 8),
				StyleBlocked.Render(gate),
				StyleNormal.Render(since))

			hint := StyleHelp.Render(fmt.Sprintf("  porch approve %s %s", builder.ID, gate))

			if selected {
				b.WriteString(StyleSelected.Render(line) + hint)
			} else {
				b.WriteString(StyleNormal.Render(line))
			}
			b.WriteString("\n")
			row++
		}
	}

	return b.String()
}
