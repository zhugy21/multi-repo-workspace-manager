package tui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	StatusSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	StatusFailedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672"))
	StatusWarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
	StatusGrayStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	StatusRunningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))

	FooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder())

	HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
)

func StatusIcon(status string) string {
	switch status {
	case "pending":
		return "○"
	case "running":
		return "⏳"
	case "success":
		return "✅"
	case "failed":
		return "✗"
	case "cancelled":
		return "⏹"
	case "skipped":
		return "◌"
	case "warning":
		return "⚠"
	default:
		return "?"
	}
}

func StatusColor(status string) lipgloss.Style {
	switch status {
	case "pending":
		return StatusGrayStyle
	case "running":
		return StatusRunningStyle
	case "success":
		return StatusSuccessStyle
	case "failed":
		return StatusFailedStyle
	case "cancelled":
		return StatusGrayStyle
	case "skipped":
		return StatusGrayStyle
	case "warning":
		return StatusWarningStyle
	default:
		return StatusGrayStyle
	}
}
