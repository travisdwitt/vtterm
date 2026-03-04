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

type inputMode int

const (
	modeNormal   inputMode = iota
	modeDPending           // received 'd', waiting for 's'
	modeDrawMenu           // draw menu: l=line, b=box
	modeDrawLine           // drawing a line
	modeDrawBox            // drawing a box
	modeMove               // moving a shape
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

	mode        inputMode
	overlay     map[[2]int]rune // committed drawn characters (world coords)
	pending     map[[2]int]rune // in-progress shape
	lineStarted bool
	boxAnchorX  int
	boxAnchorY  int
	boxW        int
	boxH        int

	moving      map[[2]int]rune // characters being moved (at original positions)
	moveOrigin  map[[2]int]rune // snapshot for cancel/restore
	moveOffsetX int
	moveOffsetY int
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
		raw = grid.RenderFlatHex(m.table.Width, m.table.Height)
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
		switch m.mode {
		case modeDPending:
			return m.updateDPending(message)
		case modeDrawMenu:
			return m.updateDrawMenu(message)
		case modeDrawLine:
			return m.updateDrawLine(message)
		case modeDrawBox:
			return m.updateDrawBox(message)
		case modeMove:
			return m.updateMove(message)
		default:
			return m.updateNormal(message)
		}
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

	case "d":
		if hasGrid {
			m.mode = modeDPending
		}

	case "m":
		if hasGrid {
			wx := m.cursorX + m.panX
			wy := m.cursorY + m.panY
			if _, ok := m.overlay[[2]int{wx, wy}]; ok {
				m.moving = m.floodSelectOverlay(wx, wy)
				// Remove selected chars from overlay
				for k := range m.moving {
					delete(m.overlay, k)
				}
				// Snapshot for cancel
				m.moveOrigin = make(map[[2]int]rune, len(m.moving))
				for k, v := range m.moving {
					m.moveOrigin[k] = v
				}
				m.moveOffsetX = 0
				m.moveOffsetY = 0
				m.mode = modeMove
			}
		}

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

func (m Model) updateDPending(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if message.String() == "s" {
		m.mode = modeDrawMenu
		m.panMode = false
		return m, nil
	}
	m.mode = modeNormal
	return m.updateNormal(message)
}

func (m Model) updateDrawMenu(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "l":
		m.mode = modeDrawLine
		m.pending = make(map[[2]int]rune)
		m.lineStarted = false
	case "b":
		m.mode = modeDrawBox
		m.pending = make(map[[2]int]rune)
		m.boxAnchorX = m.cursorX + m.panX
		m.boxAnchorY = m.cursorY + m.panY
		m.boxW = 1
		m.boxH = 1
		m.rebuildBoxPending()
	case "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m Model) updateDrawLine(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "h", "left":
		m.depositLineChar(-1, 0)
		m.clampCursor()
	case "l", "right":
		m.depositLineChar(1, 0)
		m.clampCursor()
	case "k", "up":
		m.depositLineChar(0, -1)
		m.clampCursor()
	case "j", "down":
		m.depositLineChar(0, 1)
		m.clampCursor()
	case "ctrl+s":
		m.commitPending()
		m.mode = modeNormal
	case "esc":
		m.pending = nil
		m.mode = modeNormal
	}
	return m, nil
}

func (m Model) updateDrawBox(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "l", "right":
		m.boxW++
		m.rebuildBoxPending()
	case "h", "left":
		if m.boxW > 1 {
			m.boxW--
			m.rebuildBoxPending()
		}
	case "j", "down":
		m.boxH++
		m.rebuildBoxPending()
	case "k", "up":
		if m.boxH > 1 {
			m.boxH--
			m.rebuildBoxPending()
		}
	case "ctrl+s":
		m.commitPending()
		m.mode = modeNormal
	case "esc":
		m.pending = nil
		m.mode = modeNormal
	}
	return m, nil
}

func (m Model) updateMove(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "h", "left":
		m.moveOffsetX--
	case "l", "right":
		m.moveOffsetX++
	case "k", "up":
		m.moveOffsetY--
	case "j", "down":
		m.moveOffsetY++
	case "enter":
		// Commit: write moved chars into overlay at offset positions
		if m.overlay == nil {
			m.overlay = make(map[[2]int]rune)
		}
		for k, v := range m.moving {
			m.overlay[[2]int{k[0] + m.moveOffsetX, k[1] + m.moveOffsetY}] = v
		}
		m.moving = nil
		m.moveOrigin = nil
		m.mode = modeNormal
	case "esc":
		// Cancel: restore original positions
		if m.overlay == nil {
			m.overlay = make(map[[2]int]rune)
		}
		for k, v := range m.moveOrigin {
			m.overlay[k] = v
		}
		m.moving = nil
		m.moveOrigin = nil
		m.mode = modeNormal
	}
	return m, nil
}

func (m *Model) floodSelectOverlay(startX, startY int) map[[2]int]rune {
	result := make(map[[2]int]rune)
	queue := [][2]int{{startX, startY}}
	visited := make(map[[2]int]bool)

	for len(queue) > 0 {
		pos := queue[0]
		queue = queue[1:]
		if visited[pos] {
			continue
		}
		visited[pos] = true

		r, ok := m.overlay[pos]
		if !ok {
			continue
		}
		result[pos] = r

		// 4-directional neighbors
		for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			next := [2]int{pos[0] + d[0], pos[1] + d[1]}
			if !visited[next] {
				queue = append(queue, next)
			}
		}
	}
	return result
}

func (m *Model) depositLineChar(dx, dy int) {
	maxX := m.width - 1
	if maxX < 0 {
		maxX = 0
	}
	maxY := m.height - 2
	if maxY < 0 {
		maxY = 0
	}

	newSX := m.cursorX + dx
	newSY := m.cursorY + dy
	if newSX < 0 || newSX > maxX || newSY < 0 || newSY > maxY {
		return // would be clamped, skip
	}

	wx := m.cursorX + m.panX
	wy := m.cursorY + m.panY

	var ch rune
	if dx != 0 {
		ch = '-'
	} else {
		ch = '|'
	}

	if !m.lineStarted {
		m.pending[[2]int{wx, wy}] = ch
		m.lineStarted = true
	}

	// Upgrade previous position to corner if direction changed
	prev, hasPrev := m.pending[[2]int{wx, wy}]
	if hasPrev {
		if (prev == '-' && dy != 0) || (prev == '|' && dx != 0) {
			m.pending[[2]int{wx, wy}] = '+'
		}
	}

	newWX := wx + dx
	newWY := wy + dy

	// Check intersection with existing characters
	existing, hasExisting := m.pending[[2]int{newWX, newWY}]
	if !hasExisting {
		existing, hasExisting = m.overlay[[2]int{newWX, newWY}]
	}
	if hasExisting && ((existing == '-' && dy != 0) || (existing == '|' && dx != 0) || existing == '+') {
		ch = '+'
	}

	m.pending[[2]int{newWX, newWY}] = ch

	m.cursorX = newSX
	m.cursorY = newSY
}

func (m *Model) rebuildBoxPending() {
	m.pending = make(map[[2]int]rune)
	ax, ay := m.boxAnchorX, m.boxAnchorY
	w, h := m.boxW, m.boxH

	// Draw edges
	for x := range w {
		m.pending[[2]int{ax + x, ay}] = '-'         // top
		m.pending[[2]int{ax + x, ay + h - 1}] = '-' // bottom
	}
	for y := range h {
		m.pending[[2]int{ax, ay + y}] = '|'         // left
		m.pending[[2]int{ax + w - 1, ay + y}] = '|' // right
	}

	// Corners (written last to overwrite edges)
	m.pending[[2]int{ax, ay}] = '+'
	m.pending[[2]int{ax + w - 1, ay}] = '+'
	m.pending[[2]int{ax, ay + h - 1}] = '+'
	m.pending[[2]int{ax + w - 1, ay + h - 1}] = '+'
}

func (m *Model) commitPending() {
	if m.overlay == nil {
		m.overlay = make(map[[2]int]rune)
	}
	for k, v := range m.pending {
		m.overlay[k] = v
	}
	m.pending = nil
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
	switch m.table.GridType {
	case table.GridTypeGrid:
		return grid.SquareCellCenter
	case table.GridTypeHex:
		return grid.FlatHexCellCenter
	default:
		return grid.FlatHexCellCenter
	}
}

func (m *Model) detectCell(wx, wy int) (col, row int, ok bool) {
	cols, rows := m.table.Width, m.table.Height
	switch m.table.GridType {
	case table.GridTypeGrid:
		return grid.DetectSquareCell(wx, wy, cols, rows)
	case table.GridTypeHex:
		return grid.DetectFlatHexCell(wx, wy, cols, rows)
	default:
		return grid.DetectFlatHexCell(wx, wy, cols, rows)
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
		runes, isOverlay := m.worldLine(wy, vpW)

		for i, r := range runes {
			ch := string(r)
			if sy == m.cursorY && i == m.cursorX {
				sb.WriteString(styles.CursorStyle.Render(ch))
			} else if isOverlay[i] {
				sb.WriteString(styles.OverlayStyle.Render(ch))
			} else {
				sb.WriteString(styles.GridStyle.Render(ch))
			}
		}

		if sy < vpH-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// worldLine extracts a horizontal slice of width w from gridLines at world row wy,
// starting at panX offset. Returns runes and a parallel bool slice indicating which
// positions are overlay characters (pending, moving, or committed overlay).
func (m Model) worldLine(wy, w int) ([]rune, []bool) {
	var gridRunes []rune
	if wy >= 0 && wy < len(m.gridLines) {
		gridRunes = []rune(m.gridLines[wy])
	}
	result := make([]rune, w)
	overlay := make([]bool, w)
	for i := range w {
		wx := i + m.panX
		coord := [2]int{wx, wy}

		if r, ok := m.pending[coord]; ok {
			result[i] = r
			overlay[i] = true
		} else if m.moving != nil {
			// Check if a moving char lands here after offset
			origCoord := [2]int{wx - m.moveOffsetX, wy - m.moveOffsetY}
			if r, ok := m.moving[origCoord]; ok {
				result[i] = r
				overlay[i] = true
			} else if r, ok := m.overlay[coord]; ok {
				result[i] = r
				overlay[i] = true
			} else if wx >= 0 && wx < len(gridRunes) {
				result[i] = gridRunes[wx]
			} else {
				result[i] = ' '
			}
		} else if r, ok := m.overlay[coord]; ok {
			result[i] = r
			overlay[i] = true
		} else if wx >= 0 && wx < len(gridRunes) {
			result[i] = gridRunes[wx]
		} else {
			result[i] = ' '
		}
	}
	return result, overlay
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
		switch m.mode {
		case modeDPending:
			left += "  " + styles.Highlight.Render("d-")
		case modeDrawMenu:
			left += "  " + styles.Highlight.Render("DRAW: l=line b=box")
		case modeDrawLine:
			left += "  " + styles.Highlight.Render("LINE")
		case modeDrawBox:
			left += "  " + styles.Highlight.Render(fmt.Sprintf("BOX %dx%d", m.boxW, m.boxH))
		case modeMove:
			left += "  " + styles.Highlight.Render("MOVE")
		}
	}
	if m.statusMsg != "" {
		if left != "" {
			left += "  "
		}
		left += m.statusMsg
	}

	var right string
	switch m.mode {
	case modeDPending:
		right = styles.Subtle.Render("s: draw shape")
	case modeDrawMenu:
		right = styles.Subtle.Render("l: line  b: box  esc: cancel")
	case modeDrawLine:
		right = styles.Subtle.Render("hjkl: draw  ctrl+s: commit  esc: cancel")
	case modeDrawBox:
		right = styles.Subtle.Render("hjkl: resize  ctrl+s: commit  esc: cancel")
	case modeMove:
		right = styles.Subtle.Render("hjkl: move  enter: place  esc: cancel")
	default:
		right = styles.Subtle.Render("hjkl: move  tab: next cell  z: pan  d: draw  q: quit  s: save  S: export")
	}

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
