package views

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/traviswitt/vtterm/internal/editor"
	"github.com/traviswitt/vtterm/internal/msg"
	"github.com/traviswitt/vtterm/internal/styles"
	"github.com/traviswitt/vtterm/internal/table"
)

type tokenScreenMode int

const (
	tsMenu tokenScreenMode = iota
	tsEdit
)

type TokenScreenModel struct {
	lib   *table.TokenLibrary
	width int

	mode             tokenScreenMode
	cursor           int
	items            []tokenMenuItem
	createMode       bool
	folderMode       bool
	confirmDelete    bool
	expandedFolders  map[string]bool
	nameInput        textinput.Model
	statusMsg        string

	editIdx int
	editor  editor.Editor
}

func NewTokenScreen(lib *table.TokenLibrary, width int) TokenScreenModel {
	m := TokenScreenModel{lib: lib, width: width, expandedFolders: make(map[string]bool)}
	m.items = buildTokenMenu(lib, m.expandedFolders)
	return m
}

func (m TokenScreenModel) Init() tea.Cmd {
	return nil
}

func (m TokenScreenModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil
	case tsSaveResultMsg:
		if message.err != nil {
			m.statusMsg = styles.Error.Render("Save failed: " + message.err.Error())
		} else {
			m.statusMsg = ""
		}
		return m, nil
	case tea.KeyPressMsg:
		switch m.mode {
		case tsEdit:
			return m.updateEdit(message)
		default:
			if m.createMode || m.folderMode {
				return m.updateInput(message)
			}
			return m.updateMenu(message)
		}
	}
	return m, nil
}

func (m TokenScreenModel) updateMenu(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirmDelete {
		switch message.String() {
		case "y", "enter":
			if m.cursor >= 0 && m.cursor < len(m.items) {
				item := m.items[m.cursor]
				if item.kind == tmToken {
					deleteTokenDef(m.lib, item.tokenIdx)
				} else if item.kind == tmFolder {
					deleteFolder(m.lib, item.folder)
				}
				m.items = buildTokenMenu(m.lib, m.expandedFolders)
				if m.cursor >= len(m.items) {
					m.cursor = len(m.items) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			m.confirmDelete = false
			return m, tsSaveLibCmd(m.lib)
		case "n", "esc":
			m.confirmDelete = false
		}
		return m, nil
	}
	switch message.String() {
	case "j", "down":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.items) {
			item := m.items[m.cursor]
			if item.kind == tmFolder {
				m.expandedFolders[item.folder] = !m.expandedFolders[item.folder]
				m.items = buildTokenMenu(m.lib, m.expandedFolders)
				if m.cursor >= len(m.items) {
					m.cursor = len(m.items) - 1
				}
				return m, nil
			}
		}
		m.statusMsg = styles.Error.Render("Can't add token to a table since no table is open.")
		return m, clearAfter(3 * time.Second)
	case "n":
		ti := textinput.New()
		ti.Placeholder = "token name"
		ti.CharLimit = 64
		m.nameInput = ti
		m.createMode = true
		return m, m.nameInput.Focus()
	case "f":
		ti := textinput.New()
		ti.Placeholder = "folder name"
		ti.CharLimit = 64
		m.nameInput = ti
		m.folderMode = true
		return m, m.nameInput.Focus()
	case "e":
		if m.cursor >= 0 && m.cursor < len(m.items) {
			item := m.items[m.cursor]
			if item.kind == tmToken {
				m.editIdx = item.tokenIdx
				m.editor.Begin(m.lib.Defs[item.tokenIdx].Properties)
				m.mode = tsEdit
			}
		}
	case "d":
		if m.cursor >= 0 && m.cursor < len(m.items) {
			m.confirmDelete = true
		}
	case "esc":
		return m, func() tea.Msg { return msg.GoToMainMenu{} }
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m TokenScreenModel) updateInput(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "enter":
		name := m.nameInput.Value()
		if name == "" {
			m.createMode = false
			m.folderMode = false
			return m, nil
		}
		if m.createMode {
			folder := ""
			if m.cursor >= 0 && m.cursor < len(m.items) {
				item := m.items[m.cursor]
				if item.kind == tmFolder {
					folder = item.folder
				} else if item.folder != "" {
					folder = item.folder
				}
			}
			m.lib.Defs = append(m.lib.Defs, table.TokenDef{
				ID:     table.NewTokenID(),
				Folder: folder,
				Properties: []table.TokenProperty{
					{Key: "Name", Value: name},
				},
			})
			m.createMode = false
		} else if m.folderMode {
			m.lib.Folders = append(m.lib.Folders, name)
			m.folderMode = false
		}
		m.items = buildTokenMenu(m.lib, m.expandedFolders)
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
		}
		return m, tsSaveLibCmd(m.lib)
	case "esc":
		m.createMode = false
		m.folderMode = false
		return m, nil
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(message)
	return m, cmd
}

func (m TokenScreenModel) updateEdit(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	done, cancelled := m.editor.HandleKey(message)
	if done {
		m.lib.Defs[m.editIdx].Properties = m.editor.Commit()
		m.mode = tsMenu
		m.items = buildTokenMenu(m.lib, m.expandedFolders)
		return m, tsSaveLibCmd(m.lib)
	}
	if cancelled {
		m.mode = tsMenu
		return m, nil
	}
	return m, nil
}

func (m TokenScreenModel) View() tea.View {
	if m.mode == tsEdit {
		return tea.NewView(m.viewEdit())
	}
	return tea.NewView(m.viewMenu())
}

func (m TokenScreenModel) viewMenu() string {
	var sb strings.Builder
	sb.WriteString(m.topBar())
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf("\n  %s\n\n", styles.Title.Render("Token Library")))

	if m.confirmDelete && m.cursor >= 0 && m.cursor < len(m.items) {
		item := m.items[m.cursor]
		name := "this item"
		if item.kind == tmToken && item.tokenIdx < len(m.lib.Defs) {
			if len(m.lib.Defs[item.tokenIdx].Properties) > 0 {
				name = m.lib.Defs[item.tokenIdx].Properties[0].Value
			}
		} else if item.kind == tmFolder {
			name = "folder [" + item.folder + "]"
		}
		sb.WriteString(fmt.Sprintf("  Delete %s?\n\n", name))
		sb.WriteString(styles.Subtle.Render("  y/enter: yes  n/esc: no"))
		return sb.String()
	}

	if m.createMode {
		sb.WriteString("  New token: ")
		sb.WriteString(m.nameInput.View())
		sb.WriteByte('\n')
		return sb.String()
	}
	if m.folderMode {
		sb.WriteString("  New folder: ")
		sb.WriteString(m.nameInput.View())
		sb.WriteByte('\n')
		return sb.String()
	}

	if len(m.items) == 0 {
		sb.WriteString("  No tokens yet. Press 'n' to create one.\n")
		return sb.String()
	}

	for i, item := range m.items {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		if item.kind == tmFolder {
			arrow := "> "
			if m.expandedFolders[item.folder] {
				arrow = "v "
			}
			sb.WriteString(prefix + styles.Highlight.Render(arrow+"["+item.folder+"]") + "\n")
		} else {
			td := m.lib.Defs[item.tokenIdx]
			name := "???"
			if len(td.Properties) > 0 && td.Properties[0].Value != "" {
				name = td.Properties[0].Value
			}
			indent := ""
			if item.folder != "" {
				indent = "  "
			}
			sb.WriteString(prefix + indent + name + "\n")
		}
	}
	return sb.String()
}

func (m TokenScreenModel) viewEdit() string {
	var sb strings.Builder
	sb.WriteString(m.topBar())
	sb.WriteByte('\n')

	if m.editIdx >= 0 && m.editIdx < len(m.lib.Defs) {
		td := m.lib.Defs[m.editIdx]
		name := "Token"
		if len(td.Properties) > 0 && td.Properties[0].Value != "" {
			name = td.Properties[0].Value
		}
		sb.WriteString(fmt.Sprintf("\n  %s\n\n", styles.Title.Render("Edit: "+name)))
	} else {
		sb.WriteString(fmt.Sprintf("\n  %s\n\n", styles.Title.Render("Edit Token")))
	}

	sb.WriteString(m.editor.View())
	return sb.String()
}

func (m TokenScreenModel) topBar() string {
	var left string
	switch m.mode {
	case tsEdit:
		left = styles.Highlight.Render("TOKEN EDIT")
	default:
		left = styles.Highlight.Render("TOKENS")
	}
	if m.statusMsg != "" {
		left += "  " + m.statusMsg
	}

	var right string
	switch m.mode {
	case tsEdit:
		right = styles.Subtle.Render("type Key: Value  enter: newline  ctrl+s: save  esc: cancel")
	default:
		if m.createMode {
			right = styles.Subtle.Render("type name  enter: create  esc: cancel")
		} else if m.folderMode {
			right = styles.Subtle.Render("type folder name  enter: create  esc: cancel")
		} else {
			right = styles.Subtle.Render("j/k: navigate  enter: place  n: new  e: edit  d: delete  f: folder  esc: back")
		}
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := m.width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

type tsSaveResultMsg struct{ err error }

func tsSaveLibCmd(lib *table.TokenLibrary) tea.Cmd {
	return func() tea.Msg {
		err := table.SaveTokenLibrary(lib)
		return tsSaveResultMsg{err: err}
	}
}
