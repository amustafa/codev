package tui

import "fmt"

type StatusBar struct {
	Connected     bool
	WorkspaceName string
	WorkspacePath string
	TowerVersion  string
}

func (s StatusBar) View(width int) string {
	var connIcon string
	if s.Connected {
		connIcon = StyleStatusConnected.Render("● connected")
	} else {
		connIcon = StyleStatusDisconnected.Render("● disconnected")
	}

	ws := s.WorkspaceName
	if ws == "" {
		ws = "no workspace"
	}

	ver := ""
	if s.TowerVersion != "" {
		ver = fmt.Sprintf(" v%s", s.TowerVersion)
	}

	left := connIcon
	center := fmt.Sprintf(" %s%s ", ws, ver)
	help := StyleHelp.Render("[tab] view  [n] new shell  [a] architect  [W] all tabs  [w] workspace  [s] send  [r/R] refresh  [q] quit")

	leftLen := len("● connected")
	centerLen := len(center)
	rightLen := len("[tab] view  [n] new shell  [a] architect  [W] all tabs  [w] workspace  [s] send  [r/R] refresh  [q] quit")

	gap := width - leftLen - centerLen - rightLen
	if gap < 0 {
		gap = 1
	}
	padding := ""
	for i := 0; i < gap/2; i++ {
		padding += " "
	}

	return StyleStatusBar.Width(width).Render(
		left + padding + center + padding + help,
	)
}
