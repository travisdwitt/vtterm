package app

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/traviswitt/vtterm/internal/msg"
	"github.com/traviswitt/vtterm/internal/styles"
	"github.com/traviswitt/vtterm/internal/table"
)

type loadModel struct {
	files         []string
	cursor        int
	err           string
	confirmDelete bool
}

func newLoadScreen() loadModel {
	files, err := table.ListSaved()
	m := loadModel{files: files}
	if err != nil {
		m.err = err.Error()
	}
	return m
}

func (m loadModel) Init() tea.Cmd {
	return nil
}

type loadedTableMsg struct {
	table *table.Table
	err   error
}

type deleteResultMsg struct {
	err error
}

func (m loadModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case loadedTableMsg:
		if message.err != nil {
			m.err = message.err.Error()
			return m, nil
		}
		return m, func() tea.Msg { return msg.GoToTableView{Table: *message.table} }
	case deleteResultMsg:
		if message.err != nil {
			m.err = message.err.Error()
			return m, nil
		}
		// Refresh file list
		files, err := table.ListSaved()
		m.files = files
		if err != nil {
			m.err = err.Error()
		}
		if m.cursor >= len(m.files) {
			m.cursor = len(m.files) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.confirmDelete {
			return m.updateConfirmDelete(message)
		}
		switch message.String() {
		case "esc":
			return m, func() tea.Msg { return msg.GoToMainMenu{} }
		case "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.files) > 0 {
				return m, loadTableCmd(m.files[m.cursor])
			}
		case "d":
			if len(m.files) > 0 {
				m.confirmDelete = true
			}
		}
	}
	return m, nil
}

func (m loadModel) updateConfirmDelete(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "y", "enter":
		name := m.files[m.cursor]
		m.confirmDelete = false
		return m, func() tea.Msg {
			err := table.DeleteSaved(name)
			return deleteResultMsg{err: err}
		}
	case "n", "esc":
		m.confirmDelete = false
	}
	return m, nil
}

func loadTableCmd(name string) tea.Cmd {
	return func() tea.Msg {
		t, err := table.Load(filepath.Join(table.SaveDir(), name))
		return loadedTableMsg{table: t, err: err}
	}
}

func (m loadModel) View() tea.View {
	if m.confirmDelete && len(m.files) > 0 {
		name := m.files[m.cursor]
		s := fmt.Sprintf("\n\n%s\n\n", styles.Title.Render("Delete Table?"))
		s += fmt.Sprintf("  Delete %s?\n  This cannot be undone.\n\n", name)
		s += styles.Subtle.Render("  y/enter: yes  n/esc: no")
		return tea.NewView(s)
	}

	s := styles.Title.Render("Load Table") + "\n\n"

	if m.err != "" {
		s += styles.Error.Render(m.err) + "\n\n"
	}

	if len(m.files) == 0 {
		s += styles.Subtle.Render("No saved tables found.") + "\n"
	} else {
		for i, f := range m.files {
			if m.cursor == i {
				s += styles.Highlight.Render("> "+f) + "\n"
			} else {
				s += "  " + f + "\n"
			}
		}
	}

	s += "\n" + styles.Subtle.Render("↑/↓: navigate  enter: load  d: delete  esc: back")
	return tea.NewView(s)
}
