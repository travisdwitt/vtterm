package styles

import "charm.land/lipgloss/v2"

var (
	DarkGray = lipgloss.Color("240")

	GridStyle = lipgloss.NewStyle().Foreground(DarkGray)

	Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))

	Subtle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	Highlight = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	Status = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))

	Error = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	TokenStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	TokenDisabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	CursorStyle = lipgloss.NewStyle().Reverse(true)
)
