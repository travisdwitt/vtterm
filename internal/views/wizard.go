package views

import (
	"fmt"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"github.com/traviswitt/vtterm/internal/msg"
	"github.com/traviswitt/vtterm/internal/styles"
	"github.com/traviswitt/vtterm/internal/table"
)

type wizardStep int

const (
	stepGridType wizardStep = iota
	stepHeight
	stepWidth
	stepDone
)

type WizardModel struct {
	step        wizardStep
	gridType    table.GridType
	height      int
	width       int
	textInput   textinput.Model
	err         string
	gridChoices []table.GridType
	gridCursor  int
}

func NewWizard() WizardModel {
	ti := textinput.New()
	ti.Placeholder = "e.g. 10"
	ti.CharLimit = 3
	ti.Validate = validateNumeric

	return WizardModel{
		step:        stepGridType,
		textInput:   ti,
		gridChoices: []table.GridType{table.GridTypeGrid, table.GridTypeHex, table.GridTypeNone},
	}
}

func (m WizardModel) Init() tea.Cmd {
	return nil
}

func (m WizardModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyPressMsg:
		switch message.String() {
		case "esc":
			return m, func() tea.Msg { return msg.GoToMainMenu{} }
		case "ctrl+c":
			return m, tea.Quit
		}

		switch m.step {
		case stepGridType:
			return m.updateGridType(message)
		case stepHeight:
			return m.updateDimension(message, stepWidth)
		case stepWidth:
			return m.updateDimension(message, stepDone)
		}
	}

	if m.step == stepHeight || m.step == stepWidth {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(message)
		return m, cmd
	}

	return m, nil
}

func (m WizardModel) updateGridType(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "up", "k":
		if m.gridCursor > 0 {
			m.gridCursor--
		}
	case "down", "j":
		if m.gridCursor < len(m.gridChoices)-1 {
			m.gridCursor++
		}
	case "enter":
		m.gridType = m.gridChoices[m.gridCursor]
		if m.gridType == table.GridTypeNone {
			return m, m.finish()
		} else {
			m.step = stepHeight
			m.textInput.Prompt = "Height (rows): "
			return m, m.textInput.Focus()
		}
	}
	return m, nil
}

func (m WizardModel) updateDimension(message tea.KeyPressMsg, next wizardStep) (tea.Model, tea.Cmd) {
	if message.String() != "enter" {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(message)
		return m, cmd
	}

	val, err := strconv.Atoi(m.textInput.Value())
	if err != nil || val < 1 || val > 200 {
		m.err = "Enter a number between 1 and 200"
		return m, nil
	}
	m.err = ""

	if m.step == stepHeight {
		m.height = val
	} else {
		m.width = val
	}

	if next == stepDone {
		return m, m.finish()
	}

	m.step = next
	m.textInput.SetValue("")
	m.textInput.Prompt = "Width (columns): "
	return m, nil
}

func (m WizardModel) finish() tea.Cmd {
	t := table.Table{
		Name:      "Untitled",
		GridType:  m.gridType,
		Width:     m.width,
		Height:    m.height,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return func() tea.Msg { return msg.GoToTableView{Table: t} }
}

func (m WizardModel) View() tea.View {
	s := styles.Title.Render("New Table") + "\n\n"

	switch m.step {
	case stepGridType:
		s += "Select grid type:\n\n"
		labels := []string{"Grid", "Hexes", "None"}
		descs := []string{"Square grid", "Hexagonal grid", "Blank canvas"}
		for i, label := range labels {
			if m.gridCursor == i {
				s += styles.Highlight.Render(fmt.Sprintf("> %-8s", label))
			} else {
				s += fmt.Sprintf("  %-8s", label)
			}
			s += styles.Subtle.Render("  "+descs[i]) + "\n"
		}
	case stepHeight, stepWidth:
		s += m.textInput.View() + "\n"
		if m.err != "" {
			s += styles.Error.Render(m.err) + "\n"
		}
	}

	s += "\n" + styles.Subtle.Render("↑/↓: navigate  enter: confirm  esc: back")
	return tea.NewView(s)
}

func validateNumeric(s string) error {
	if s == "" {
		return nil
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return fmt.Errorf("digits only")
		}
	}
	return nil
}
