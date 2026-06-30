package tui

import "github.com/charmbracelet/lipgloss"

var (
	ColorGreen  = lipgloss.Color("#22c55e")
	ColorRed    = lipgloss.Color("#ef4444")
	ColorYellow = lipgloss.Color("#eab308")
	ColorBlue   = lipgloss.Color("#3b82f6")
	ColorGray   = lipgloss.Color("#6b7280")
	ColorWhite  = lipgloss.Color("#f9fafb")
	ColorDim    = lipgloss.Color("#374151")
	ColorCyan   = lipgloss.Color("#06b6d4")

	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			MarginBottom(1)

	StyleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorBlue).
			Padding(0, 1)

	StyleTabInactive = lipgloss.NewStyle().
				Foreground(ColorGray).
				Padding(0, 1)

	StyleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite)

	StyleNormal = lipgloss.NewStyle().
			Foreground(ColorGray)

	StyleBlocked = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	StyleActive = lipgloss.NewStyle().
			Foreground(ColorGreen)

	StylePRReady = lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)

	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorYellow)

	StyleBadge = lipgloss.NewStyle().
			Padding(0, 1)

	StyleBadgeSpec = StyleBadge.
			Foreground(ColorGreen)

	StyleBadgeMissing = StyleBadge.
				Foreground(ColorDim)

	StyleStatusBar = lipgloss.NewStyle().
			Foreground(ColorGray).
			Background(lipgloss.Color("#1f2937")).
			Padding(0, 1)

	StyleStatusConnected = lipgloss.NewStyle().
				Foreground(ColorGreen).
				Bold(true)

	StyleStatusDisconnected = lipgloss.NewStyle().
				Foreground(ColorRed).
				Bold(true)

	StyleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorCyan).
			Padding(1, 2).
			Width(60)

	StyleModalTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			MarginBottom(1)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorRed)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorGreen)

	StyleDivider = lipgloss.NewStyle().
			Foreground(ColorDim)
)
