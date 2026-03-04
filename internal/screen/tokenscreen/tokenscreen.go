package tokenscreen

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/traviswitt/vtterm/internal/msg"
	"github.com/traviswitt/vtterm/internal/styles"
	"github.com/traviswitt/vtterm/internal/table"
)

type mode int

const (
	modeMenu mode = iota
	modeEdit
)

type tokenMenuItemKind int

const (
	tmFolder tokenMenuItemKind = iota
	tmToken
)

type tokenMenuItem struct {
	kind     tokenMenuItemKind
	folder   string
	tokenIdx int // index into lib.Defs (tmToken only)
}

type Model struct {
	lib   *table.TokenLibrary
	width int

	mode             mode
	cursor           int
	items            []tokenMenuItem
	createMode       bool
	folderMode       bool
	confirmDelete    bool
	expandedFolders map[string]bool
	nameInput        textinput.Model
	statusMsg        string

	// edit mode
	editIdx    int
	editLines  [][]rune
	editCurRow int
	editCurCol int
}

type clearStatusMsg struct{}

func New(lib *table.TokenLibrary, width int) Model {
	m := Model{lib: lib, width: width, expandedFolders: make(map[string]bool)}
	m.rebuildMenu()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil
	case saveResultMsg:
		if message.err != nil {
			m.statusMsg = styles.Error.Render("Save failed: " + message.err.Error())
		} else {
			m.statusMsg = ""
		}
		return m, nil
	case tea.KeyPressMsg:
		switch m.mode {
		case modeEdit:
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

func (m Model) updateMenu(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirmDelete {
		switch message.String() {
		case "y", "enter":
			if m.cursor >= 0 && m.cursor < len(m.items) {
				item := m.items[m.cursor]
				if item.kind == tmToken {
					m.deleteToken(item.tokenIdx)
				} else if item.kind == tmFolder {
					m.deleteFolder(item.folder)
				}
				m.rebuildMenu()
				if m.cursor >= len(m.items) {
					m.cursor = len(m.items) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			m.confirmDelete = false
			return m, saveLibCmd(m.lib)
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
				m.rebuildMenu()
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
				m.beginEdit(item.tokenIdx)
				m.mode = modeEdit
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

func (m Model) updateInput(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		m.rebuildMenu()
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
		}
		return m, saveLibCmd(m.lib)
	case "esc":
		m.createMode = false
		m.folderMode = false
		return m, nil
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(message)
	return m, cmd
}

func (m Model) updateEdit(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := message.String()
	switch key {
	case "enter":
		line := m.editLines[m.editCurRow]
		before := make([]rune, m.editCurCol)
		copy(before, line[:m.editCurCol])
		after := make([]rune, len(line)-m.editCurCol)
		copy(after, line[m.editCurCol:])
		m.editLines[m.editCurRow] = before
		newLines := make([][]rune, len(m.editLines)+1)
		copy(newLines, m.editLines[:m.editCurRow+1])
		newLines[m.editCurRow+1] = after
		copy(newLines[m.editCurRow+2:], m.editLines[m.editCurRow+1:])
		m.editLines = newLines
		m.editCurRow++
		m.editCurCol = 0
	case "backspace":
		if m.editCurCol > 0 {
			line := m.editLines[m.editCurRow]
			m.editLines[m.editCurRow] = append(line[:m.editCurCol-1], line[m.editCurCol:]...)
			m.editCurCol--
		} else if m.editCurRow > 0 {
			prevLine := m.editLines[m.editCurRow-1]
			joinCol := len(prevLine)
			m.editLines[m.editCurRow-1] = append(prevLine, m.editLines[m.editCurRow]...)
			m.editLines = append(m.editLines[:m.editCurRow], m.editLines[m.editCurRow+1:]...)
			m.editCurRow--
			m.editCurCol = joinCol
		}
	case "delete":
		line := m.editLines[m.editCurRow]
		if m.editCurCol < len(line) {
			m.editLines[m.editCurRow] = append(line[:m.editCurCol], line[m.editCurCol+1:]...)
		} else if m.editCurRow < len(m.editLines)-1 {
			m.editLines[m.editCurRow] = append(line, m.editLines[m.editCurRow+1]...)
			m.editLines = append(m.editLines[:m.editCurRow+1], m.editLines[m.editCurRow+2:]...)
		}
	case "left":
		if m.editCurCol > 0 {
			m.editCurCol--
		} else if m.editCurRow > 0 {
			m.editCurRow--
			m.editCurCol = len(m.editLines[m.editCurRow])
		}
	case "right":
		line := m.editLines[m.editCurRow]
		if m.editCurCol < len(line) {
			m.editCurCol++
		} else if m.editCurRow < len(m.editLines)-1 {
			m.editCurRow++
			m.editCurCol = 0
		}
	case "up":
		if m.editCurRow > 0 {
			m.editCurRow--
			if m.editCurCol > len(m.editLines[m.editCurRow]) {
				m.editCurCol = len(m.editLines[m.editCurRow])
			}
		}
	case "down":
		if m.editCurRow < len(m.editLines)-1 {
			m.editCurRow++
			if m.editCurCol > len(m.editLines[m.editCurRow]) {
				m.editCurCol = len(m.editLines[m.editCurRow])
			}
		}
	case "ctrl+s":
		m.commitEdit()
		m.mode = modeMenu
		m.rebuildMenu()
		return m, saveLibCmd(m.lib)
	case "esc":
		m.mode = modeMenu
		return m, nil
	default:
		if len(message.Text) > 0 {
			r := []rune(message.Text)
			line := m.editLines[m.editCurRow]
			newLine := make([]rune, 0, len(line)+len(r))
			newLine = append(newLine, line[:m.editCurCol]...)
			newLine = append(newLine, r...)
			newLine = append(newLine, line[m.editCurCol:]...)
			m.editLines[m.editCurRow] = newLine
			m.editCurCol += len(r)
		}
	}
	return m, nil
}

func (m *Model) rebuildMenu() {
	m.items = nil
	for i, td := range m.lib.Defs {
		if td.Folder == "" {
			m.items = append(m.items, tokenMenuItem{kind: tmToken, tokenIdx: i})
		}
	}
	folders := make([]string, len(m.lib.Folders))
	copy(folders, m.lib.Folders)
	sort.Strings(folders)
	for _, f := range folders {
		m.items = append(m.items, tokenMenuItem{kind: tmFolder, folder: f})
		if m.expandedFolders[f] {
			for i, td := range m.lib.Defs {
				if td.Folder == f {
					m.items = append(m.items, tokenMenuItem{kind: tmToken, folder: f, tokenIdx: i})
				}
			}
		}
	}
}

func (m *Model) deleteToken(idx int) {
	m.lib.Defs = append(m.lib.Defs[:idx], m.lib.Defs[idx+1:]...)
}

func (m *Model) deleteFolder(name string) {
	for i := range m.lib.Defs {
		if m.lib.Defs[i].Folder == name {
			m.lib.Defs[i].Folder = ""
		}
	}
	folders := m.lib.Folders[:0]
	for _, f := range m.lib.Folders {
		if f != name {
			folders = append(folders, f)
		}
	}
	m.lib.Folders = folders
}

func (m *Model) beginEdit(defIdx int) {
	m.editIdx = defIdx
	td := m.lib.Defs[defIdx]
	m.editLines = nil
	for _, prop := range td.Properties {
		m.editLines = append(m.editLines, []rune(prop.Key+": "+prop.Value))
	}
	m.editLines = append(m.editLines, []rune{})
	m.editCurRow = len(m.editLines) - 1
	m.editCurCol = 0
}

func (m *Model) commitEdit() {
	var props []table.TokenProperty
	for _, line := range m.editLines {
		s := string(line)
		if idx := strings.Index(s, ": "); idx >= 0 {
			props = append(props, table.TokenProperty{Key: s[:idx], Value: s[idx+2:]})
		} else if len(s) > 0 {
			props = append(props, table.TokenProperty{Key: s, Value: ""})
		}
	}
	m.lib.Defs[m.editIdx].Properties = props
}

func (m Model) View() tea.View {
	if m.mode == modeEdit {
		return tea.NewView(m.viewEdit())
	}
	return tea.NewView(m.viewMenu())
}

func (m Model) viewMenu() string {
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

func (m Model) viewEdit() string {
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

	for row, line := range m.editLines {
		sb.WriteString("  ")
		for col, ch := range line {
			s := string(ch)
			if row == m.editCurRow && col == m.editCurCol {
				sb.WriteString(styles.CursorStyle.Render(s))
			} else {
				sb.WriteString(s)
			}
		}
		if row == m.editCurRow && m.editCurCol == len(line) {
			sb.WriteString(styles.CursorStyle.Render(" "))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (m Model) topBar() string {
	var left string
	switch m.mode {
	case modeEdit:
		left = styles.Highlight.Render("TOKEN EDIT")
	default:
		left = styles.Highlight.Render("TOKENS")
	}
	if m.statusMsg != "" {
		left += "  " + m.statusMsg
	}

	var right string
	switch m.mode {
	case modeEdit:
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

type saveResultMsg struct{ err error }

func saveLibCmd(lib *table.TokenLibrary) tea.Cmd {
	return func() tea.Msg {
		err := table.SaveTokenLibrary(lib)
		return saveResultMsg{err: err}
	}
}

func clearAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}
