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

	OverlayStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	CursorStyle = lipgloss.NewStyle().Reverse(true)

	DialogBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("170")).
			Padding(1, 2).
			Width(40)
)
