package tui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary   = lipgloss.Color("#7C3AED")
	ColorSecondary = lipgloss.Color("#06B6D4")
	ColorSuccess   = lipgloss.Color("#10B981")
	ColorWarning   = lipgloss.Color("#F59E0B")
	ColorError     = lipgloss.Color("#EF4444")
	ColorMuted     = lipgloss.Color("#6B7280")
	ColorBg        = lipgloss.Color("#1F2937")

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	StyleSubtitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorError)

	StyleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	StyleRepoName = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	StyleHash = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StyleAuthor = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StyleDate = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleIssue = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StylePR = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	StyleMerged = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A855F7"))

	StyleStatusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#374151")).
			Padding(0, 1).
			Width(80)
)
