package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cluesmith-ai/codev-tui/internal/api"
)

type BacklogView struct {
	items  []api.OverviewBacklogItem
	groups []backlogGroup
	cursor int
	total  int
}

type backlogGroup struct {
	area  string
	items []api.OverviewBacklogItem
}

func NewBacklogView() BacklogView {
	return BacklogView{}
}

func (v *BacklogView) SetItems(items []api.OverviewBacklogItem) {
	v.items = items
	v.groups = groupByArea(items)
	v.total = len(items)
	if v.cursor >= v.total {
		v.cursor = max(0, v.total-1)
	}
}

func (v *BacklogView) Up() {
	if v.cursor > 0 {
		v.cursor--
	}
}

func (v *BacklogView) Down() {
	if v.cursor < v.total-1 {
		v.cursor++
	}
}

func (v *BacklogView) Selected() *api.OverviewBacklogItem {
	row := 0
	for _, g := range v.groups {
		for i := range g.items {
			if row == v.cursor {
				return &g.items[i]
			}
			row++
		}
	}
	return nil
}

func (v BacklogView) View(width int) string {
	if v.total == 0 {
		return StyleNormal.Render("  No backlog items")
	}

	var b strings.Builder
	row := 0

	for _, g := range v.groups {
		b.WriteString(StyleHeader.Render(fmt.Sprintf("  %s (%d)", g.area, len(g.items))))
		b.WriteString("\n")

		for _, item := range g.items {
			selected := row == v.cursor
			prefix := "  "
			if selected {
				prefix = "▶ "
			}

			badges := statusBadges(item.HasSpec, item.HasPlan, item.HasReview, item.HasBuilder)
			line := fmt.Sprintf("%s#%-5s %-40s %s", prefix,
				item.ID,
				truncate(item.Title, 40),
				badges)

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

	return b.String()
}

func statusBadges(spec, plan, review, builder bool) string {
	badge := func(label string, has bool) string {
		if has {
			return StyleBadgeSpec.Render("✓" + label)
		}
		return StyleBadgeMissing.Render("·" + label)
	}
	return badge("S", spec) + " " + badge("P", plan) + " " + badge("R", review) + " " + badge("B", builder)
}

func groupByArea(items []api.OverviewBacklogItem) []backlogGroup {
	groups := make(map[string][]api.OverviewBacklogItem)
	for _, item := range items {
		area := item.Area
		if area == "" {
			area = "Uncategorized"
		}
		groups[area] = append(groups[area], item)
	}

	var result []backlogGroup
	for area, items := range groups {
		result = append(result, backlogGroup{area: area, items: items})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].area < result[j].area
	})
	return result
}
