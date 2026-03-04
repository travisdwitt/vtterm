package mainmenu

import (
	tea "charm.land/bubbletea/v2"
	"github.com/traviswitt/vtterm/internal/msg"
	"github.com/traviswitt/vtterm/internal/styles"
)

var choices = []string{"New Table", "Load Table", "Tokens", "Exit"}

type Model struct {
	cursor int
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyPressMsg:
		switch message.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(choices)-1 {
				m.cursor++
			}
		case "enter":
			switch m.cursor {
			case 0:
				return m, func() tea.Msg { return msg.GoToWizard{} }
			case 1:
				return m, func() tea.Msg { return msg.GoToLoad{} }
			case 2:
				return m, func() tea.Msg { return msg.GoToTokens{} }
			case 3:
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	logo := "        __   __\n" +
		".--.--.|  |_|  |_.-----.----.--------.\n" +
		"|  |  ||   _|   _|  -__|   _|        |\n" +
		" \\___/ |____|____|_____|__| |__|__|__|"
	s := styles.Title.Render(logo) + "\n"
	s += styles.Subtle.Render("A virtual tabletop in your terminal") + "\n\n"

	for i, choice := range choices {
		if m.cursor == i {
			s += styles.Highlight.Render("[>] "+choice) + "\n"
		} else {
			s += "[ ] " + choice + "\n"
		}
	}

	s += "\n" + styles.Subtle.Render("↑/↓: navigate  enter: select")
	return tea.NewView(s)
}
