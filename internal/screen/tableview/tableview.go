package tableview

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/traviswitt/vtterm/internal/grid"
	"github.com/traviswitt/vtterm/internal/msg"
	"github.com/traviswitt/vtterm/internal/styles"
	"github.com/traviswitt/vtterm/internal/table"
)

type Model struct {
	table     table.Table
	gridLines []string // raw unstyled lines from renderer
	showQuit  bool
	statusMsg string
	width     int
	height    int
	cursorX   int // screen-space cursor X
	cursorY   int // screen-space cursor Y
	panX      int // viewport X offset
	panY      int // viewport Y offset
	panMode   bool
}

type clearStatusMsg struct{}

func New(t table.Table, w, h int) Model {
	m := Model{table: t, width: w, height: h}
	m.renderGrid()
	return m
}

func (m *Model) renderGrid() {
	var raw string
	switch m.table.GridType {
	case table.GridTypeGrid:
		raw = grid.RenderSquare(m.table.Width, m.table.Height)
	case table.GridTypeHex:
		if m.table.HexOrientation == table.HexFlatTop {
			raw = grid.RenderFlatHex(m.table.Width, m.table.Height)
		} else {
			raw = grid.RenderPointyHex(m.table.Width, m.table.Height)
		}
	case table.GridTypeNone:
		m.gridLines = nil
		return
	}
	m.gridLines = strings.Split(strings.TrimRight(raw, "\n"), "\n")
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		m.clampCursor()
	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil
	case saveResultMsg:
		if message.err != nil {
			m.statusMsg = styles.Error.Render("Save failed: " + message.err.Error())
		} else {
			m.statusMsg = styles.Status.Render("Saved!")
		}
		return m, clearAfter(2 * time.Second)
	case exportResultMsg:
		if message.err != nil {
			m.statusMsg = styles.Error.Render("Export failed: " + message.err.Error())
		} else {
			m.statusMsg = styles.Status.Render("Exported to " + message.path)
		}
		return m, clearAfter(3 * time.Second)
	case tea.KeyPressMsg:
		if m.showQuit {
			return m.updateQuitDialog(message)
		}
		return m.updateNormal(message)
	}
	return m, nil
}

func (m Model) updateNormal(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := message.String()
	hasGrid := m.table.GridType != table.GridTypeNone

	switch key {
	case "q":
		m.showQuit = true
		return m, nil
	case "s":
		return m, saveCmd(m.table)
	case "S":
		return m, exportCmd(m.table)
	case "ctrl+c":
		return m, tea.Quit

	case "z":
		if hasGrid {
			m.panMode = !m.panMode
		}
	case "esc":
		m.panMode = false

	case "h", "left":
		if hasGrid {
			if m.panMode {
				m.panX++
			} else {
				m.cursorX--
			}
		}
	case "l", "right":
		if hasGrid {
			if m.panMode {
				m.panX--
			} else {
				m.cursorX++
			}
		}
	case "k", "up":
		if hasGrid {
			if m.panMode {
				m.panY++
			} else {
				m.cursorY--
			}
		}
	case "j", "down":
		if hasGrid {
			if m.panMode {
				m.panY--
			} else {
				m.cursorY++
			}
		}

	case "H", "shift+left":
		if hasGrid {
			if m.panMode {
				m.panX += 2
			} else {
				m.cursorX -= 2
			}
		}
	case "L", "shift+right":
		if hasGrid {
			if m.panMode {
				m.panX -= 2
			} else {
				m.cursorX += 2
			}
		}
	case "K", "shift+up":
		if hasGrid {
			if m.panMode {
				m.panY += 2
			} else {
				m.cursorY -= 2
			}
		}
	case "J", "shift+down":
		if hasGrid {
			if m.panMode {
				m.panY -= 2
			} else {
				m.cursorY += 2
			}
		}

	case "tab":
		if hasGrid {
			m.handleTab(false)
		}
	case "shift+tab":
		if hasGrid {
			m.handleTab(true)
		}
	}

	if hasGrid {
		m.clampCursor()
	}
	return m, nil
}

func (m *Model) clampCursor() {
	maxX := m.width - 1
	if maxX < 0 {
		maxX = 0
	}
	maxY := m.height - 2 // 1 row reserved for top bar
	if maxY < 0 {
		maxY = 0
	}
	if m.cursorX < 0 {
		m.cursorX = 0
	}
	if m.cursorX > maxX {
		m.cursorX = maxX
	}
	if m.cursorY < 0 {
		m.cursorY = 0
	}
	if m.cursorY > maxY {
		m.cursorY = maxY
	}
}

func (m *Model) centerFunc() grid.CenterFunc {
	switch {
	case m.table.GridType == table.GridTypeGrid:
		return grid.SquareCellCenter
	case m.table.GridType == table.GridTypeHex && m.table.HexOrientation == table.HexFlatTop:
		return grid.FlatHexCellCenter
	default:
		return grid.PointyHexCellCenter
	}
}

func (m *Model) detectCell(wx, wy int) (col, row int, ok bool) {
	cols, rows := m.table.Width, m.table.Height
	switch {
	case m.table.GridType == table.GridTypeGrid:
		return grid.DetectSquareCell(wx, wy, cols, rows)
	case m.table.GridType == table.GridTypeHex && m.table.HexOrientation == table.HexFlatTop:
		return grid.DetectFlatHexCell(wx, wy, cols, rows)
	default:
		return grid.DetectPointyHexCell(wx, wy, cols, rows)
	}
}

func (m *Model) handleTab(reverse bool) {
	cols, rows := m.table.Width, m.table.Height
	if cols == 0 || rows == 0 {
		return
	}

	cfn := m.centerFunc()
	wx := m.cursorX + m.panX
	wy := m.cursorY + m.panY

	nearCol, nearRow := grid.NearestCell(wx, wy, cols, rows, cfn)
	cx, cy := cfn(nearCol, nearRow)

	// If cursor is not at the nearest cell's center, snap to it.
	if wx != cx || wy != cy {
		m.panToWorld(cx, cy)
		return
	}

	// Already at center: advance to next/previous cell in row-major order.
	idx := nearRow*cols + nearCol
	if reverse {
		idx--
		if idx < 0 {
			idx = cols*rows - 1
		}
	} else {
		idx++
		if idx >= cols*rows {
			idx = 0
		}
	}
	nextCol := idx % cols
	nextRow := idx / cols
	nx, ny := cfn(nextCol, nextRow)
	m.panToWorld(nx, ny)
}

// panToWorld moves the cursor to world position (wx, wy), adjusting pan offsets
// if needed to keep the cursor within screen bounds.
func (m *Model) panToWorld(wx, wy int) {
	m.cursorX = wx - m.panX
	m.cursorY = wy - m.panY

	maxX := m.width - 1
	if maxX < 0 {
		maxX = 0
	}
	maxY := m.height - 2
	if maxY < 0 {
		maxY = 0
	}

	if m.cursorX < 0 {
		m.panX += m.cursorX
		m.cursorX = 0
	} else if m.cursorX > maxX {
		m.panX += m.cursorX - maxX
		m.cursorX = maxX
	}

	if m.cursorY < 0 {
		m.panY += m.cursorY
		m.cursorY = 0
	} else if m.cursorY > maxY {
		m.panY += m.cursorY - maxY
		m.cursorY = maxY
	}
}

func (m Model) updateQuitDialog(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "y", "enter":
		return m, func() tea.Msg { return msg.GoToMainMenu{} }
	case "n", "esc":
		m.showQuit = false
	}
	return m, nil
}

type saveResultMsg struct{ err error }
type exportResultMsg struct {
	path string
	err  error
}

func saveCmd(t table.Table) tea.Cmd {
	return func() tea.Msg {
		err := table.Save(&t)
		return saveResultMsg{err: err}
	}
}

func exportCmd(t table.Table) tea.Cmd {
	return func() tea.Msg {
		path, err := table.Export(&t)
		return exportResultMsg{path: path, err: err}
	}
}

func clearAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func (m Model) View() tea.View {
	if m.showQuit {
		return tea.NewView(m.viewQuitDialog())
	}

	if m.table.GridType == table.GridTypeNone {
		return tea.NewView(m.topBar() + "\n")
	}

	vpH := m.height - 1 // 1 row for top bar
	if vpH < 1 {
		vpH = 1
	}
	vpW := m.width
	if vpW < 1 {
		vpW = 1
	}

	return tea.NewView(m.topBar() + "\n" + m.renderViewport(vpW, vpH))
}

func (m Model) renderViewport(vpW, vpH int) string {
	var sb strings.Builder
	for sy := range vpH {
		wy := sy + m.panY
		line := m.worldLine(wy, vpW)

		if sy == m.cursorY {
			// Split line into pre-cursor, cursor char, post-cursor.
			cx := m.cursorX
			if cx < 0 {
				cx = 0
			}
			if cx >= vpW {
				cx = vpW - 1
			}

			runes := []rune(line)
			pre := string(runes[:cx])
			ch := string(runes[cx : cx+1])
			post := string(runes[cx+1:])

			sb.WriteString(styles.GridStyle.Render(pre))
			sb.WriteString(styles.CursorStyle.Render(ch))
			sb.WriteString(styles.GridStyle.Render(post))
		} else {
			sb.WriteString(styles.GridStyle.Render(line))
		}

		if sy < vpH-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// worldLine extracts a horizontal slice of width w from gridLines at world row wy,
// starting at panX offset. Pads with spaces if outside grid bounds.
func (m Model) worldLine(wy, w int) string {
	if wy < 0 || wy >= len(m.gridLines) {
		return strings.Repeat(" ", w)
	}
	src := m.gridLines[wy]
	runes := []rune(src)
	result := make([]rune, w)
	for i := range w {
		wx := i + m.panX
		if wx >= 0 && wx < len(runes) {
			result[i] = runes[wx]
		} else {
			result[i] = ' '
		}
	}
	return string(result)
}

func (m Model) topBar() string {
	var left string
	if m.table.GridType != table.GridTypeNone {
		wx := m.cursorX + m.panX
		wy := m.cursorY + m.panY
		if col, row, ok := m.detectCell(wx, wy); ok {
			left = fmt.Sprintf("Cell: %d,%d", col, row)
		} else {
			left = fmt.Sprintf("Pos: %d,%d", wx, wy)
		}
		if m.panMode {
			left += "  " + styles.Highlight.Render("PAN")
		}
	}
	if m.statusMsg != "" {
		if left != "" {
			left += "  "
		}
		left += m.statusMsg
	}

	right := styles.Subtle.Render("hjkl: move  tab: next cell  z: pan  q: quit  s: save  S: export")

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := m.width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

func (m Model) viewQuitDialog() string {
	s := fmt.Sprintf("\n\n%s\n\n", styles.Title.Render("Quit?"))
	s += "  Are you sure you want to quit?\n"
	s += "  Unsaved changes will be lost.\n\n"
	s += styles.Subtle.Render("  y/enter: yes  n/esc: no")
	return s
}
