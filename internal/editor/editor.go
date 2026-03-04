package editor

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/traviswitt/vtterm/internal/styles"
	"github.com/traviswitt/vtterm/internal/table"
)

// Editor is a simple multi-line text editor for editing "Key: Value" property lines.
type Editor struct {
	Lines  [][]rune
	CurRow int
	CurCol int
}

// Begin initializes the editor with lines from the given token properties,
// plus a trailing blank line.
func (e *Editor) Begin(props []table.TokenProperty) {
	e.Lines = nil
	for _, prop := range props {
		e.Lines = append(e.Lines, []rune(prop.Key+": "+prop.Value))
	}
	e.Lines = append(e.Lines, []rune{})
	e.CurRow = len(e.Lines) - 1
	e.CurCol = 0
}

// Commit parses the editor lines back into token properties.
func (e *Editor) Commit() []table.TokenProperty {
	var props []table.TokenProperty
	for _, line := range e.Lines {
		s := string(line)
		if idx := strings.Index(s, ": "); idx >= 0 {
			props = append(props, table.TokenProperty{Key: s[:idx], Value: s[idx+2:]})
		} else if len(s) > 0 {
			props = append(props, table.TokenProperty{Key: s, Value: ""})
		}
	}
	return props
}

// HandleKey processes a key press. Returns done=true on ctrl+s, cancelled=true on esc.
func (e *Editor) HandleKey(msg tea.KeyPressMsg) (done, cancelled bool) {
	key := msg.String()
	switch key {
	case "enter":
		line := e.Lines[e.CurRow]
		before := make([]rune, e.CurCol)
		copy(before, line[:e.CurCol])
		after := make([]rune, len(line)-e.CurCol)
		copy(after, line[e.CurCol:])
		e.Lines[e.CurRow] = before
		newLines := make([][]rune, len(e.Lines)+1)
		copy(newLines, e.Lines[:e.CurRow+1])
		newLines[e.CurRow+1] = after
		copy(newLines[e.CurRow+2:], e.Lines[e.CurRow+1:])
		e.Lines = newLines
		e.CurRow++
		e.CurCol = 0
	case "backspace":
		if e.CurCol > 0 {
			line := e.Lines[e.CurRow]
			e.Lines[e.CurRow] = append(line[:e.CurCol-1], line[e.CurCol:]...)
			e.CurCol--
		} else if e.CurRow > 0 {
			prevLine := e.Lines[e.CurRow-1]
			joinCol := len(prevLine)
			e.Lines[e.CurRow-1] = append(prevLine, e.Lines[e.CurRow]...)
			e.Lines = append(e.Lines[:e.CurRow], e.Lines[e.CurRow+1:]...)
			e.CurRow--
			e.CurCol = joinCol
		}
	case "delete":
		line := e.Lines[e.CurRow]
		if e.CurCol < len(line) {
			e.Lines[e.CurRow] = append(line[:e.CurCol], line[e.CurCol+1:]...)
		} else if e.CurRow < len(e.Lines)-1 {
			e.Lines[e.CurRow] = append(line, e.Lines[e.CurRow+1]...)
			e.Lines = append(e.Lines[:e.CurRow+1], e.Lines[e.CurRow+2:]...)
		}
	case "left":
		if e.CurCol > 0 {
			e.CurCol--
		} else if e.CurRow > 0 {
			e.CurRow--
			e.CurCol = len(e.Lines[e.CurRow])
		}
	case "right":
		line := e.Lines[e.CurRow]
		if e.CurCol < len(line) {
			e.CurCol++
		} else if e.CurRow < len(e.Lines)-1 {
			e.CurRow++
			e.CurCol = 0
		}
	case "up":
		if e.CurRow > 0 {
			e.CurRow--
			if e.CurCol > len(e.Lines[e.CurRow]) {
				e.CurCol = len(e.Lines[e.CurRow])
			}
		}
	case "down":
		if e.CurRow < len(e.Lines)-1 {
			e.CurRow++
			if e.CurCol > len(e.Lines[e.CurRow]) {
				e.CurCol = len(e.Lines[e.CurRow])
			}
		}
	case "ctrl+s":
		return true, false
	case "esc":
		return false, true
	default:
		if len(msg.Text) > 0 {
			r := []rune(msg.Text)
			line := e.Lines[e.CurRow]
			newLine := make([]rune, 0, len(line)+len(r))
			newLine = append(newLine, line[:e.CurCol]...)
			newLine = append(newLine, r...)
			newLine = append(newLine, line[e.CurCol:]...)
			e.Lines[e.CurRow] = newLine
			e.CurCol += len(r)
		}
	}
	return false, false
}

// View renders the editor lines with cursor highlighting.
func (e *Editor) View() string {
	var sb strings.Builder
	for row, line := range e.Lines {
		sb.WriteString("  ")
		for col, ch := range line {
			s := string(ch)
			if row == e.CurRow && col == e.CurCol {
				sb.WriteString(styles.CursorStyle.Render(s))
			} else {
				sb.WriteString(s)
			}
		}
		if row == e.CurRow && e.CurCol == len(line) {
			sb.WriteString(styles.CursorStyle.Render(" "))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
