package views

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/traviswitt/vtterm/internal/editor"
	"github.com/traviswitt/vtterm/internal/grid"
	"github.com/traviswitt/vtterm/internal/msg"
	"github.com/traviswitt/vtterm/internal/styles"
	"github.com/traviswitt/vtterm/internal/table"
)

type inputMode int

const (
	modeNormal   inputMode = iota
	modeMPending           // received 'm', waiting for m/d/l
	modeDrawMenu           // draw menu: l=line, b=box
	modeDrawLine           // drawing a line
	modeDrawBox            // drawing a box
	modeText               // typing free-form text
	modeMove               // moving a shape
	modeLayer              // layer selection with +/-
	modeSaveName           // entering a name before first save
	modeTokenMenu          // browsing token library
	modeTokenEdit          // editing token properties
	modeTokenColor         // picking token color
)

var tokenColors = []struct {
	name  string
	color string
}{
	{"Orange", "214"},
	{"Red", "196"},
	{"Green", "82"},
	{"Blue", "39"},
	{"Yellow", "226"},
	{"Magenta", "201"},
	{"Cyan", "51"},
	{"White", "255"},
	{"Brown", "172"},
	{"Dark Red", "124"},
	{"Dark Green", "28"},
	{"Dark Blue", "27"},
	{"Purple", "141"},
	{"Pink", "212"},
	{"Light Blue", "117"},
	{"Gray", "248"},
}

type TableViewModel struct {
	table     table.Table
	tokenLib  *table.TokenLibrary
	gridLines []string
	showQuit  bool
	statusMsg string
	width     int
	height    int
	cursorX   int
	cursorY   int
	panX      int
	panY      int
	panMode   bool

	mode         inputMode
	prevMode     inputMode
	layers       map[int]map[[2]int]rune
	layerColors  map[int]map[[2]int]string
	currentLayer int
	pending      map[[2]int]rune
	lineStarted  bool
	boxAnchorX   int
	boxAnchorY   int
	boxW         int
	boxH         int

	textLines   [][]rune
	textAnchorX int
	textAnchorY int
	textCurRow  int
	textCurCol  int

	moving           map[[2]int]rune
	movingColors     map[[2]int]string
	moveOriginColors map[[2]int]string
	moveOrigin       map[[2]int]rune
	moveOffsetX      int
	moveOffsetY      int
	moveSourceLayer  int

	nameInput textinput.Model

	tokenMenuCursor    int
	tokenMenuItems     []tokenMenuItem
	tokenCreateMode    bool
	tokenFolderMode    bool
	tokenConfirmDelete bool
	expandedFolders    map[string]bool

	tokenEditor editor.Editor
	tokenEditIdx int

	movingTokenIdx       int
	tokenMoveOffsetX     int
	tokenMoveOffsetY     int
	tokenMoveOriginX     int
	tokenMoveOriginY     int
	tokenMoveOriginLayer int

	inspectTokenIdx int
	deleteTokenIdx  int

	colorTokenIdx         int
	colorOverlayPositions [][2]int
	colorCursor           int

	tokenNameInput textinput.Model
}

type tvSaveResultMsg struct{ err error }
type exportResultMsg struct {
	path string
	err  error
}

func NewTableView(t table.Table, lib *table.TokenLibrary, w, h int) TableViewModel {
	m := TableViewModel{table: t, tokenLib: lib, width: w, height: h}
	m.layers = make(map[int]map[[2]int]rune)
	m.layerColors = make(map[int]map[[2]int]string)
	m.movingTokenIdx = -1
	m.inspectTokenIdx = -1
	m.deleteTokenIdx = -1
	m.colorTokenIdx = -1
	m.expandedFolders = make(map[string]bool)
	m.renderGrid()
	m.loadOverlayFromTable()
	return m
}

func (m *TableViewModel) activeOverlay() map[[2]int]rune {
	ol, ok := m.layers[m.currentLayer]
	if !ok {
		ol = make(map[[2]int]rune)
		m.layers[m.currentLayer] = ol
	}
	return ol
}

func (m *TableViewModel) syncOverlayToTable() {
	var total int
	for _, ol := range m.layers {
		total += len(ol)
	}
	if total == 0 {
		m.table.Overlay = nil
		return
	}
	chars := make([]table.OverlayChar, 0, total)
	for layer, ol := range m.layers {
		lc := m.layerColors[layer]
		for k, v := range ol {
			oc := table.OverlayChar{X: k[0], Y: k[1], R: string(v), Layer: layer}
			if lc != nil {
				oc.Color = lc[k]
			}
			chars = append(chars, oc)
		}
	}
	m.table.Overlay = chars
}

func (m *TableViewModel) loadOverlayFromTable() {
	if len(m.table.Overlay) == 0 {
		return
	}
	for _, oc := range m.table.Overlay {
		runes := []rune(oc.R)
		if len(runes) == 0 {
			continue
		}
		ol, ok := m.layers[oc.Layer]
		if !ok {
			ol = make(map[[2]int]rune)
			m.layers[oc.Layer] = ol
		}
		pos := [2]int{oc.X, oc.Y}
		ol[pos] = runes[0]
		if oc.Color != "" {
			lc, ok := m.layerColors[oc.Layer]
			if !ok {
				lc = make(map[[2]int]string)
				m.layerColors[oc.Layer] = lc
			}
			lc[pos] = oc.Color
		}
	}
}

func (m *TableViewModel) renderGrid() {
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

func (m TableViewModel) Init() tea.Cmd {
	return nil
}

func (m TableViewModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		m.clampCursor()
	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil
	case tvSaveResultMsg:
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
		if m.deleteTokenIdx >= 0 {
			return m.updateDeleteTokenDialog(message)
		}
		switch m.mode {
		case modeSaveName:
			return m.updateSaveName(message)
		case modeMPending:
			return m.updateMPending(message)
		case modeLayer:
			return m.updateLayer(message)
		case modeDrawMenu:
			return m.updateDrawMenu(message)
		case modeDrawLine:
			return m.updateDrawLine(message)
		case modeDrawBox:
			return m.updateDrawBox(message)
		case modeText:
			return m.updateText(message)
		case modeMove:
			return m.updateMove(message)
		case modeTokenMenu:
			return m.updateTokenMenu(message)
		case modeTokenEdit:
			return m.updateTokenEdit(message)
		case modeTokenColor:
			return m.updateTokenColor(message)
		default:
			return m.updateNormal(message)
		}
	}
	return m, nil
}

func (m TableViewModel) updateNormal(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := message.String()
	hasGrid := m.table.GridType != table.GridTypeNone

	switch key {
	case "q":
		m.showQuit = true
		return m, nil
	case "s":
		if m.table.Name == "Untitled" {
			ti := textinput.New()
			ti.Placeholder = "table name"
			ti.CharLimit = 64
			m.nameInput = ti
			m.mode = modeSaveName
			return m, m.nameInput.Focus()
		}
		m.syncOverlayToTable()
		return m, tvSaveCmd(m.table)
	case "S":
		return m, tvExportCmd(m.table)
	case "ctrl+c":
		return m, tea.Quit

	case "m":
		if hasGrid {
			m.prevMode = modeNormal
			m.mode = modeMPending
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

	case "T":
		if hasGrid {
			m.tokenMenuItems = buildTokenMenu(m.tokenLib, m.expandedFolders)
			m.tokenMenuCursor = 0
			m.tokenCreateMode = false
			m.tokenFolderMode = false
			m.mode = modeTokenMenu
		}
	case "i":
		if hasGrid {
			if m.inspectTokenIdx >= 0 {
				m.inspectTokenIdx = -1
			} else {
				wx := m.cursorX + m.panX
				wy := m.cursorY + m.panY
				m.inspectTokenIdx = m.findTokenPlacementAt(wx, wy)
			}
		}
	case "e":
		if hasGrid {
			wx := m.cursorX + m.panX
			wy := m.cursorY + m.panY
			pi := m.findTokenPlacementAt(wx, wy)
			if pi >= 0 {
				tid := m.table.TokenPlacements[pi].TokenID
				for di := range m.tokenLib.Defs {
					if m.tokenLib.Defs[di].ID == tid {
						m.tokenEditIdx = di
						m.tokenEditor.Begin(m.tokenLib.Defs[di].Properties)
						m.mode = modeTokenEdit
						break
					}
				}
			}
		}
	case "r":
		if hasGrid {
			wx := m.cursorX + m.panX
			wy := m.cursorY + m.panY
			pi := m.findTokenPlacementAt(wx, wy)
			if pi >= 0 {
				faces := 4
				if m.table.GridType == table.GridTypeHex {
					faces = 6
				}
				m.table.TokenPlacements[pi].Facing = (m.table.TokenPlacements[pi].Facing + 1) % faces
			}
		}
	case "d":
		if hasGrid {
			wx := m.cursorX + m.panX
			wy := m.cursorY + m.panY
			pi := m.findTokenPlacementAt(wx, wy)
			if pi >= 0 {
				def := m.tokenLib.FindTokenDef(m.table.TokenPlacements[pi].TokenID)
				if def != nil {
					if def.IsDisabled() {
						props := def.Properties[:0]
						for _, p := range def.Properties {
							if p.Key != "Disabled" {
								props = append(props, p)
							}
						}
						def.Properties = props
					} else {
						def.Properties = append(def.Properties, table.TokenProperty{Key: "Disabled", Value: "true"})
					}
				}
			}
		}
	case "D":
		if hasGrid {
			wx := m.cursorX + m.panX
			wy := m.cursorY + m.panY
			pi := m.findTokenPlacementAt(wx, wy)
			if pi >= 0 {
				m.deleteTokenIdx = pi
			}
		}
	case "c":
		if hasGrid {
			wx := m.cursorX + m.panX
			wy := m.cursorY + m.panY
			pi := m.findTokenPlacementAt(wx, wy)
			if pi >= 0 {
				m.colorTokenIdx = pi
				m.colorOverlayPositions = nil
				m.colorCursor = 0
				if c := m.table.TokenPlacements[pi].Color; c != "" {
					for i, tc := range tokenColors {
						if tc.color == c {
							m.colorCursor = i
							break
						}
					}
				}
				m.mode = modeTokenColor
			} else if ol := m.layers[m.currentLayer]; ol != nil {
				if _, ok := ol[[2]int{wx, wy}]; ok {
					positions := m.floodSelectOverlayPositions(wx, wy)
					m.colorTokenIdx = -1
					m.colorOverlayPositions = positions
					m.colorCursor = 0
					if lc := m.layerColors[m.currentLayer]; lc != nil {
						if c, ok := lc[[2]int{wx, wy}]; ok {
							for i, tc := range tokenColors {
								if tc.color == c {
									m.colorCursor = i
									break
								}
							}
						}
					}
					m.mode = modeTokenColor
				}
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

func (m TableViewModel) updateSaveName(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "enter":
		name := m.nameInput.Value()
		if name == "" {
			name = "Untitled"
		}
		m.table.Name = name
		m.syncOverlayToTable()
		m.mode = modeNormal
		return m, tvSaveCmd(m.table)
	case "esc":
		m.mode = modeNormal
		return m, nil
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(message)
	return m, cmd
}

func (m TableViewModel) updateMPending(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "m":
		if m.prevMode == modeNormal {
			wx := m.cursorX + m.panX
			wy := m.cursorY + m.panY
			ti := m.findTokenPlacementAt(wx, wy)
			if ti >= 0 {
				m.movingTokenIdx = ti
				p := m.table.TokenPlacements[ti]
				m.tokenMoveOriginX = p.X
				m.tokenMoveOriginY = p.Y
				m.tokenMoveOriginLayer = p.Layer
				m.tokenMoveOffsetX = 0
				m.tokenMoveOffsetY = 0
				m.mode = modeMove
			} else {
				ol := m.activeOverlay()
				if _, ok := ol[[2]int{wx, wy}]; ok {
					m.moving = m.floodSelectOverlay(wx, wy)
					lc := m.layerColors[m.currentLayer]
					m.movingColors = make(map[[2]int]string, len(m.moving))
					for k := range m.moving {
						delete(ol, k)
						if lc != nil {
							if c, ok := lc[k]; ok {
								m.movingColors[k] = c
								delete(lc, k)
							}
						}
					}
					m.moveOrigin = make(map[[2]int]rune, len(m.moving))
					m.moveOriginColors = make(map[[2]int]string, len(m.movingColors))
					for k, v := range m.moving {
						m.moveOrigin[k] = v
					}
					for k, v := range m.movingColors {
						m.moveOriginColors[k] = v
					}
					m.moveOffsetX = 0
					m.moveOffsetY = 0
					m.moveSourceLayer = m.currentLayer
					m.mode = modeMove
				} else {
					m.mode = m.prevMode
				}
			}
		} else {
			m.mode = m.prevMode
		}
	case "d":
		if m.prevMode == modeNormal {
			m.mode = modeDrawMenu
			m.panMode = false
		} else {
			m.mode = m.prevMode
		}
	case "t":
		if m.prevMode == modeNormal {
			m.mode = modeText
			m.textAnchorX = m.cursorX + m.panX
			m.textAnchorY = m.cursorY + m.panY
			m.textLines = [][]rune{{}}
			m.textCurRow = 0
			m.textCurCol = 0
			m.pending = make(map[[2]int]rune)
		} else {
			m.mode = m.prevMode
		}
	case "l":
		m.mode = modeLayer
	case "esc":
		m.mode = m.prevMode
	default:
		m.mode = m.prevMode
	}
	return m, nil
}

func (m TableViewModel) updateLayer(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "+", "=":
		m.currentLayer++
	case "-":
		if m.currentLayer > 0 {
			m.currentLayer--
		}
	case "enter":
		if m.prevMode == modeMove {
			if m.movingTokenIdx >= 0 {
				m.table.TokenPlacements[m.movingTokenIdx].X = m.tokenMoveOriginX + m.tokenMoveOffsetX
				m.table.TokenPlacements[m.movingTokenIdx].Y = m.tokenMoveOriginY + m.tokenMoveOffsetY
				m.table.TokenPlacements[m.movingTokenIdx].Layer = m.currentLayer
				m.movingTokenIdx = -1
				m.mode = modeNormal
			} else {
				ol := m.activeOverlay()
				lc, ok := m.layerColors[m.currentLayer]
				if !ok {
					lc = make(map[[2]int]string)
					m.layerColors[m.currentLayer] = lc
				}
				for k, v := range m.moving {
					newPos := [2]int{k[0] + m.moveOffsetX, k[1] + m.moveOffsetY}
					ol[newPos] = v
					if c, ok := m.movingColors[k]; ok {
						lc[newPos] = c
					}
				}
				m.moving = nil
				m.movingColors = nil
				m.moveOrigin = nil
				m.moveOriginColors = nil
				m.mode = modeNormal
			}
		} else {
			m.mode = m.prevMode
		}
	case "esc", "l":
		m.mode = m.prevMode
	}
	return m, nil
}

func (m TableViewModel) updateDrawMenu(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m TableViewModel) updateDrawLine(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m TableViewModel) updateDrawBox(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m TableViewModel) updateText(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := message.String()
	switch key {
	case "enter":
		line := m.textLines[m.textCurRow]
		before := make([]rune, m.textCurCol)
		copy(before, line[:m.textCurCol])
		after := make([]rune, len(line)-m.textCurCol)
		copy(after, line[m.textCurCol:])
		m.textLines[m.textCurRow] = before
		newLines := make([][]rune, len(m.textLines)+1)
		copy(newLines, m.textLines[:m.textCurRow+1])
		newLines[m.textCurRow+1] = after
		copy(newLines[m.textCurRow+2:], m.textLines[m.textCurRow+1:])
		m.textLines = newLines
		m.textCurRow++
		m.textCurCol = 0
	case "backspace":
		if m.textCurCol > 0 {
			line := m.textLines[m.textCurRow]
			m.textLines[m.textCurRow] = append(line[:m.textCurCol-1], line[m.textCurCol:]...)
			m.textCurCol--
		} else if m.textCurRow > 0 {
			prevLine := m.textLines[m.textCurRow-1]
			joinCol := len(prevLine)
			m.textLines[m.textCurRow-1] = append(prevLine, m.textLines[m.textCurRow]...)
			m.textLines = append(m.textLines[:m.textCurRow], m.textLines[m.textCurRow+1:]...)
			m.textCurRow--
			m.textCurCol = joinCol
		}
	case "delete":
		line := m.textLines[m.textCurRow]
		if m.textCurCol < len(line) {
			m.textLines[m.textCurRow] = append(line[:m.textCurCol], line[m.textCurCol+1:]...)
		} else if m.textCurRow < len(m.textLines)-1 {
			m.textLines[m.textCurRow] = append(line, m.textLines[m.textCurRow+1]...)
			m.textLines = append(m.textLines[:m.textCurRow+1], m.textLines[m.textCurRow+2:]...)
		}
	case "left":
		if m.textCurCol > 0 {
			m.textCurCol--
		} else if m.textCurRow > 0 {
			m.textCurRow--
			m.textCurCol = len(m.textLines[m.textCurRow])
		}
	case "right":
		line := m.textLines[m.textCurRow]
		if m.textCurCol < len(line) {
			m.textCurCol++
		} else if m.textCurRow < len(m.textLines)-1 {
			m.textCurRow++
			m.textCurCol = 0
		}
	case "up":
		if m.textCurRow > 0 {
			m.textCurRow--
			if m.textCurCol > len(m.textLines[m.textCurRow]) {
				m.textCurCol = len(m.textLines[m.textCurRow])
			}
		}
	case "down":
		if m.textCurRow < len(m.textLines)-1 {
			m.textCurRow++
			if m.textCurCol > len(m.textLines[m.textCurRow]) {
				m.textCurCol = len(m.textLines[m.textCurRow])
			}
		}
	case "ctrl+s":
		m.commitPending()
		m.mode = modeNormal
		return m, nil
	case "esc":
		m.pending = nil
		m.mode = modeNormal
		return m, nil
	default:
		if len(message.Text) > 0 {
			r := []rune(message.Text)
			line := m.textLines[m.textCurRow]
			newLine := make([]rune, 0, len(line)+len(r))
			newLine = append(newLine, line[:m.textCurCol]...)
			newLine = append(newLine, r...)
			newLine = append(newLine, line[m.textCurCol:]...)
			m.textLines[m.textCurRow] = newLine
			m.textCurCol += len(r)
		}
	}
	m.rebuildTextPending()
	m.panToWorld(m.textAnchorX+m.textCurCol, m.textAnchorY+m.textCurRow)
	return m, nil
}

func (m *TableViewModel) rebuildTextPending() {
	m.pending = make(map[[2]int]rune)
	for row, line := range m.textLines {
		for col, ch := range line {
			m.pending[[2]int{m.textAnchorX + col, m.textAnchorY + row}] = ch
		}
	}
}

func (m TableViewModel) updateMove(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.movingTokenIdx >= 0 {
		return m.updateMoveToken(message)
	}
	switch message.String() {
	case "h", "left":
		m.moveOffsetX--
	case "l", "right":
		m.moveOffsetX++
	case "k", "up":
		m.moveOffsetY--
	case "j", "down":
		m.moveOffsetY++
	case "m":
		m.prevMode = modeMove
		m.mode = modeMPending
	case "enter":
		ol := m.activeOverlay()
		lc, ok := m.layerColors[m.currentLayer]
		if !ok {
			lc = make(map[[2]int]string)
			m.layerColors[m.currentLayer] = lc
		}
		for k, v := range m.moving {
			newPos := [2]int{k[0] + m.moveOffsetX, k[1] + m.moveOffsetY}
			ol[newPos] = v
			if c, ok := m.movingColors[k]; ok {
				lc[newPos] = c
			}
		}
		m.moving = nil
		m.movingColors = nil
		m.moveOrigin = nil
		m.moveOriginColors = nil
		m.mode = modeNormal
	case "esc":
		srcOl, ok := m.layers[m.moveSourceLayer]
		if !ok {
			srcOl = make(map[[2]int]rune)
			m.layers[m.moveSourceLayer] = srcOl
		}
		for k, v := range m.moveOrigin {
			srcOl[k] = v
		}
		if len(m.moveOriginColors) > 0 {
			srcLc, ok := m.layerColors[m.moveSourceLayer]
			if !ok {
				srcLc = make(map[[2]int]string)
				m.layerColors[m.moveSourceLayer] = srcLc
			}
			for k, v := range m.moveOriginColors {
				srcLc[k] = v
			}
		}
		m.moving = nil
		m.movingColors = nil
		m.moveOrigin = nil
		m.moveOriginColors = nil
		m.mode = modeNormal
	}
	return m, nil
}

func (m TableViewModel) updateMoveToken(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "h", "left":
		m.tokenMoveOffsetX--
	case "l", "right":
		m.tokenMoveOffsetX++
	case "k", "up":
		m.tokenMoveOffsetY--
	case "j", "down":
		m.tokenMoveOffsetY++
	case "m":
		m.prevMode = modeMove
		m.mode = modeMPending
	case "enter":
		m.table.TokenPlacements[m.movingTokenIdx].X = m.tokenMoveOriginX + m.tokenMoveOffsetX
		m.table.TokenPlacements[m.movingTokenIdx].Y = m.tokenMoveOriginY + m.tokenMoveOffsetY
		m.table.TokenPlacements[m.movingTokenIdx].Layer = m.currentLayer
		m.movingTokenIdx = -1
		m.mode = modeNormal
	case "esc":
		m.table.TokenPlacements[m.movingTokenIdx].X = m.tokenMoveOriginX
		m.table.TokenPlacements[m.movingTokenIdx].Y = m.tokenMoveOriginY
		m.table.TokenPlacements[m.movingTokenIdx].Layer = m.tokenMoveOriginLayer
		m.movingTokenIdx = -1
		m.mode = modeNormal
	}
	return m, nil
}

func (m TableViewModel) floodSelectOverlayPositions(startX, startY int) [][2]int {
	ol := m.layers[m.currentLayer]
	if ol == nil {
		return nil
	}
	var result [][2]int
	queue := [][2]int{{startX, startY}}
	visited := make(map[[2]int]bool)

	for len(queue) > 0 {
		pos := queue[0]
		queue = queue[1:]
		if visited[pos] {
			continue
		}
		visited[pos] = true

		if _, ok := ol[pos]; !ok {
			continue
		}
		result = append(result, pos)

		for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			next := [2]int{pos[0] + d[0], pos[1] + d[1]}
			if !visited[next] {
				queue = append(queue, next)
			}
		}
	}
	return result
}

func (m *TableViewModel) floodSelectOverlay(startX, startY int) map[[2]int]rune {
	positions := m.floodSelectOverlayPositions(startX, startY)
	ol := m.activeOverlay()
	result := make(map[[2]int]rune, len(positions))
	for _, pos := range positions {
		result[pos] = ol[pos]
	}
	return result
}

func (m *TableViewModel) depositLineChar(dx, dy int) {
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
		return
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
		existing, hasExisting = m.activeOverlay()[[2]int{newWX, newWY}]
	}
	if hasExisting && ((existing == '-' && dy != 0) || (existing == '|' && dx != 0) || existing == '+') {
		ch = '+'
	}

	m.pending[[2]int{newWX, newWY}] = ch

	m.cursorX = newSX
	m.cursorY = newSY
}

func (m *TableViewModel) rebuildBoxPending() {
	m.pending = make(map[[2]int]rune)
	ax, ay := m.boxAnchorX, m.boxAnchorY
	w, h := m.boxW, m.boxH

	for x := range w {
		m.pending[[2]int{ax + x, ay}] = '-'
		m.pending[[2]int{ax + x, ay + h - 1}] = '-'
	}
	for y := range h {
		m.pending[[2]int{ax, ay + y}] = '|'
		m.pending[[2]int{ax + w - 1, ay + y}] = '|'
	}

	m.pending[[2]int{ax, ay}] = '+'
	m.pending[[2]int{ax + w - 1, ay}] = '+'
	m.pending[[2]int{ax, ay + h - 1}] = '+'
	m.pending[[2]int{ax + w - 1, ay + h - 1}] = '+'
}

func (m *TableViewModel) commitPending() {
	ol := m.activeOverlay()
	for k, v := range m.pending {
		ol[k] = v
	}
	m.pending = nil
}

func (m *TableViewModel) clampCursor() {
	maxX := m.width - 1
	if maxX < 0 {
		maxX = 0
	}
	maxY := m.height - 2
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

func (m *TableViewModel) centerFunc() grid.CenterFunc {
	switch m.table.GridType {
	case table.GridTypeGrid:
		return grid.SquareCellCenter
	case table.GridTypeHex:
		return grid.FlatHexCellCenter
	default:
		return grid.FlatHexCellCenter
	}
}

func (m *TableViewModel) detectCell(wx, wy int) (col, row int, ok bool) {
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

func (m *TableViewModel) handleTab(reverse bool) {
	cols, rows := m.table.Width, m.table.Height
	if cols == 0 || rows == 0 {
		return
	}

	cfn := m.centerFunc()
	wx := m.cursorX + m.panX
	wy := m.cursorY + m.panY

	nearCol, nearRow := grid.NearestCell(wx, wy, cols, rows, cfn)
	cx, cy := cfn(nearCol, nearRow)

	if wx != cx || wy != cy {
		m.panToWorld(cx, cy)
		return
	}

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

func (m *TableViewModel) panToWorld(wx, wy int) {
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

func (m TableViewModel) updateQuitDialog(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "y", "enter":
		return m, func() tea.Msg { return msg.GoToMainMenu{} }
	case "n", "esc":
		m.showQuit = false
	}
	return m, nil
}

func (m TableViewModel) updateDeleteTokenDialog(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "y", "enter":
		if m.deleteTokenIdx >= 0 && m.deleteTokenIdx < len(m.table.TokenPlacements) {
			m.table.TokenPlacements = append(
				m.table.TokenPlacements[:m.deleteTokenIdx],
				m.table.TokenPlacements[m.deleteTokenIdx+1:]...,
			)
			if m.inspectTokenIdx == m.deleteTokenIdx {
				m.inspectTokenIdx = -1
			} else if m.inspectTokenIdx > m.deleteTokenIdx {
				m.inspectTokenIdx--
			}
		}
		m.deleteTokenIdx = -1
	case "n", "esc":
		m.deleteTokenIdx = -1
	}
	return m, nil
}

func tvSaveCmd(t table.Table) tea.Cmd {
	return func() tea.Msg {
		err := table.Save(&t)
		return tvSaveResultMsg{err: err}
	}
}

func tvExportCmd(t table.Table) tea.Cmd {
	return func() tea.Msg {
		path, err := table.Export(&t)
		return exportResultMsg{path: path, err: err}
	}
}

func (m TableViewModel) resolveTokenColor(p table.TokenPlacement) string {
	def := m.tokenLib.FindTokenDef(p.TokenID)
	if def != nil && def.IsDisabled() {
		return "disabled"
	}
	if p.Color != "" {
		return p.Color
	}
	return "214"
}

func (m TableViewModel) findTokenPlacementAt(wx, wy int) int {
	for i, p := range m.table.TokenPlacements {
		if p.Layer != m.currentLayer {
			continue
		}
		if wx >= p.X && wx < p.X+5 && wy >= p.Y && wy < p.Y+3 {
			return i
		}
	}
	return -1
}

func (m TableViewModel) updateTokenMenu(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.tokenConfirmDelete {
		switch message.String() {
		case "y", "enter":
			if m.tokenMenuCursor >= 0 && m.tokenMenuCursor < len(m.tokenMenuItems) {
				item := m.tokenMenuItems[m.tokenMenuCursor]
				if item.kind == tmToken {
					m.tvDeleteToken(item.tokenIdx)
				} else if item.kind == tmFolder {
					deleteFolder(m.tokenLib, item.folder)
				}
				m.tokenMenuItems = buildTokenMenu(m.tokenLib, m.expandedFolders)
				if m.tokenMenuCursor >= len(m.tokenMenuItems) {
					m.tokenMenuCursor = len(m.tokenMenuItems) - 1
				}
				if m.tokenMenuCursor < 0 {
					m.tokenMenuCursor = 0
				}
			}
			m.tokenConfirmDelete = false
		case "n", "esc":
			m.tokenConfirmDelete = false
		}
		return m, nil
	}
	if m.tokenCreateMode || m.tokenFolderMode {
		return m.updateTokenMenuInput(message)
	}
	switch message.String() {
	case "j", "down":
		if m.tokenMenuCursor < len(m.tokenMenuItems)-1 {
			m.tokenMenuCursor++
		}
	case "k", "up":
		if m.tokenMenuCursor > 0 {
			m.tokenMenuCursor--
		}
	case "enter":
		if m.tokenMenuCursor >= 0 && m.tokenMenuCursor < len(m.tokenMenuItems) {
			item := m.tokenMenuItems[m.tokenMenuCursor]
			if item.kind == tmFolder {
				m.expandedFolders[item.folder] = !m.expandedFolders[item.folder]
				m.tokenMenuItems = buildTokenMenu(m.tokenLib, m.expandedFolders)
				if m.tokenMenuCursor >= len(m.tokenMenuItems) {
					m.tokenMenuCursor = len(m.tokenMenuItems) - 1
				}
			} else if item.kind == tmToken {
				td := m.tokenLib.Defs[item.tokenIdx]
				wx := m.cursorX + m.panX
				wy := m.cursorY + m.panY
				m.table.TokenPlacements = append(m.table.TokenPlacements, table.TokenPlacement{
					TokenID: td.ID,
					X:       wx - 2,
					Y:       wy - 1,
					Layer:   m.currentLayer,
				})
				m.mode = modeNormal
			}
		}
	case "n":
		ti := textinput.New()
		ti.Placeholder = "token name"
		ti.CharLimit = 64
		m.tokenNameInput = ti
		m.tokenCreateMode = true
		return m, m.tokenNameInput.Focus()
	case "f":
		ti := textinput.New()
		ti.Placeholder = "folder name"
		ti.CharLimit = 64
		m.tokenNameInput = ti
		m.tokenFolderMode = true
		return m, m.tokenNameInput.Focus()
	case "e":
		if m.tokenMenuCursor >= 0 && m.tokenMenuCursor < len(m.tokenMenuItems) {
			item := m.tokenMenuItems[m.tokenMenuCursor]
			if item.kind == tmToken {
				m.tokenEditIdx = item.tokenIdx
				m.tokenEditor.Begin(m.tokenLib.Defs[item.tokenIdx].Properties)
				m.mode = modeTokenEdit
			}
		}
	case "d":
		if m.tokenMenuCursor >= 0 && m.tokenMenuCursor < len(m.tokenMenuItems) {
			m.tokenConfirmDelete = true
		}
	case "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m TableViewModel) updateTokenMenuInput(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "enter":
		name := m.tokenNameInput.Value()
		if name == "" {
			m.tokenCreateMode = false
			m.tokenFolderMode = false
			return m, nil
		}
		if m.tokenCreateMode {
			folder := ""
			if m.tokenMenuCursor >= 0 && m.tokenMenuCursor < len(m.tokenMenuItems) {
				item := m.tokenMenuItems[m.tokenMenuCursor]
				if item.kind == tmFolder {
					folder = item.folder
				} else if item.folder != "" {
					folder = item.folder
				}
			}
			m.tokenLib.Defs = append(m.tokenLib.Defs, table.TokenDef{
				ID:     table.NewTokenID(),
				Folder: folder,
				Properties: []table.TokenProperty{
					{Key: "Name", Value: name},
				},
			})
			m.tokenCreateMode = false
		} else if m.tokenFolderMode {
			m.tokenLib.Folders = append(m.tokenLib.Folders, name)
			m.tokenFolderMode = false
		}
		m.tokenMenuItems = buildTokenMenu(m.tokenLib, m.expandedFolders)
		if len(m.tokenMenuItems) > 0 {
			m.tokenMenuCursor = len(m.tokenMenuItems) - 1
		}
		return m, nil
	case "esc":
		m.tokenCreateMode = false
		m.tokenFolderMode = false
		return m, nil
	}
	var cmd tea.Cmd
	m.tokenNameInput, cmd = m.tokenNameInput.Update(message)
	return m, cmd
}

func (m TableViewModel) updateTokenEdit(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	done, cancelled := m.tokenEditor.HandleKey(message)
	if done {
		m.tokenLib.Defs[m.tokenEditIdx].Properties = m.tokenEditor.Commit()
		m.mode = modeNormal
		return m, nil
	}
	if cancelled {
		m.mode = modeNormal
		return m, nil
	}
	return m, nil
}

func (m TableViewModel) updateTokenColor(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "j", "down":
		if m.colorCursor < len(tokenColors)-1 {
			m.colorCursor++
		}
	case "k", "up":
		if m.colorCursor > 0 {
			m.colorCursor--
		}
	case "enter":
		if m.colorOverlayPositions != nil {
			lc, ok := m.layerColors[m.currentLayer]
			if !ok {
				lc = make(map[[2]int]string)
				m.layerColors[m.currentLayer] = lc
			}
			for _, pos := range m.colorOverlayPositions {
				lc[pos] = tokenColors[m.colorCursor].color
			}
			m.colorOverlayPositions = nil
		} else if m.colorTokenIdx >= 0 && m.colorTokenIdx < len(m.table.TokenPlacements) {
			m.table.TokenPlacements[m.colorTokenIdx].Color = tokenColors[m.colorCursor].color
		}
		m.colorTokenIdx = -1
		m.mode = modeNormal
	case "esc":
		m.colorTokenIdx = -1
		m.colorOverlayPositions = nil
		m.mode = modeNormal
	}
	return m, nil
}

// tvDeleteToken removes a token def and all its placements from the table.
func (m *TableViewModel) tvDeleteToken(idx int) {
	tid := m.tokenLib.Defs[idx].ID
	deleteTokenDef(m.tokenLib, idx)
	placements := m.table.TokenPlacements[:0]
	for _, p := range m.table.TokenPlacements {
		if p.TokenID != tid {
			placements = append(placements, p)
		}
	}
	m.table.TokenPlacements = placements
	m.inspectTokenIdx = -1
}

func (m TableViewModel) View() tea.View {
	if m.showQuit {
		return tea.NewView(m.viewQuitDialog())
	}
	if m.deleteTokenIdx >= 0 {
		return tea.NewView(m.viewDeleteTokenDialog())
	}
	if m.mode == modeTokenMenu {
		return tea.NewView(m.viewTokenMenu())
	}
	if m.mode == modeTokenEdit {
		return tea.NewView(m.viewTokenEdit())
	}
	if m.mode == modeTokenColor {
		return tea.NewView(m.viewTokenColor())
	}

	if m.table.GridType == table.GridTypeNone {
		return tea.NewView(m.topBar() + "\n")
	}

	vpH := m.height - 1
	if vpH < 1 {
		vpH = 1
	}
	vpW := m.width
	if vpW < 1 {
		vpW = 1
	}

	return tea.NewView(m.topBar() + "\n" + m.renderViewport(vpW, vpH))
}

func (m TableViewModel) renderViewport(vpW, vpH int) string {
	var sb strings.Builder
	for sy := range vpH {
		wy := sy + m.panY
		runes, overlayColor, tokenColor := m.worldLine(wy, vpW)

		for i, r := range runes {
			ch := string(r)
			if sy == m.cursorY && i == m.cursorX {
				sb.WriteString(styles.CursorStyle.Render(ch))
			} else if tokenColor[i] == "disabled" {
				sb.WriteString(styles.TokenDisabledStyle.Render(ch))
			} else if tokenColor[i] != "" {
				sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(tokenColor[i])).Render(ch))
			} else if overlayColor[i] != "" {
				sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(overlayColor[i])).Render(ch))
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

func (m TableViewModel) worldLine(wy, w int) ([]rune, []string, []string) {
	var gridRunes []rune
	if wy >= 0 && wy < len(m.gridLines) {
		gridRunes = []rune(m.gridLines[wy])
	}
	result := make([]rune, w)
	overlayColor := make([]string, w)
	tokenColor := make([]string, w)

	for i := range w {
		wx := i + m.panX
		if wx >= 0 && wx < len(gridRunes) {
			result[i] = gridRunes[wx]
		} else {
			result[i] = ' '
		}
	}

	ol := m.layers[m.currentLayer]
	lc := m.layerColors[m.currentLayer]
	if ol != nil {
		for i := range w {
			wx := i + m.panX
			pos := [2]int{wx, wy}
			if r, ok := ol[pos]; ok {
				result[i] = r
				if lc != nil {
					if c, ok := lc[pos]; ok {
						overlayColor[i] = c
					} else {
						overlayColor[i] = "255"
					}
				} else {
					overlayColor[i] = "255"
				}
			}
		}
	}

	for pi, p := range m.table.TokenPlacements {
		if p.Layer != m.currentLayer {
			continue
		}
		if pi == m.movingTokenIdx {
			continue
		}
		tc := m.resolveTokenColor(p)
		box := m.tokenBoxRunes(p)
		for coord, r := range box {
			sx := coord[0] - m.panX
			if sx >= 0 && sx < w && coord[1] == wy {
				result[sx] = r
				tokenColor[sx] = tc
				overlayColor[sx] = ""
			}
		}
	}

	if m.movingTokenIdx >= 0 && m.movingTokenIdx < len(m.table.TokenPlacements) {
		p := m.table.TokenPlacements[m.movingTokenIdx]
		tc := m.resolveTokenColor(p)
		movedP := table.TokenPlacement{
			TokenID: p.TokenID,
			X:       m.tokenMoveOriginX + m.tokenMoveOffsetX,
			Y:       m.tokenMoveOriginY + m.tokenMoveOffsetY,
			Layer:   m.currentLayer,
			Facing:  p.Facing,
		}
		box := m.tokenBoxRunes(movedP)
		for coord, r := range box {
			sx := coord[0] - m.panX
			if sx >= 0 && sx < w && coord[1] == wy {
				result[sx] = r
				tokenColor[sx] = tc
				overlayColor[sx] = ""
			}
		}
	}

	if m.inspectTokenIdx >= 0 && m.inspectTokenIdx < len(m.table.TokenPlacements) {
		p := m.table.TokenPlacements[m.inspectTokenIdx]
		def := m.tokenLib.FindTokenDef(p.TokenID)
		if def != nil {
			tooltipX := p.X + 6
			for li, prop := range def.Properties {
				tooltipY := p.Y + li
				if tooltipY != wy {
					continue
				}
				line := []rune(prop.Key + ": " + prop.Value)
				for ci, r := range line {
					sx := tooltipX + ci - m.panX
					if sx >= 0 && sx < w {
						result[sx] = r
						tokenColor[sx] = "214"
						overlayColor[sx] = ""
					}
				}
			}
		}
	}

	if m.moving != nil {
		for i := range w {
			wx := i + m.panX
			origCoord := [2]int{wx - m.moveOffsetX, wy - m.moveOffsetY}
			if r, ok := m.moving[origCoord]; ok {
				result[i] = r
				if c, ok := m.movingColors[origCoord]; ok {
					overlayColor[i] = c
				} else {
					overlayColor[i] = "255"
				}
				tokenColor[i] = ""
			}
		}
	}

	for i := range w {
		wx := i + m.panX
		if r, ok := m.pending[[2]int{wx, wy}]; ok {
			result[i] = r
			overlayColor[i] = "255"
			tokenColor[i] = ""
		}
	}

	return result, overlayColor, tokenColor
}

func (m TableViewModel) tokenBoxRunes(p table.TokenPlacement) map[[2]int]rune {
	def := m.tokenLib.FindTokenDef(p.TokenID)
	label := "???"
	if def != nil {
		label = def.DisplayLabel()
	}
	labelRunes := []rune(label)

	box := make(map[[2]int]rune)
	box[[2]int{p.X, p.Y}] = '+'
	box[[2]int{p.X + 1, p.Y}] = '-'
	box[[2]int{p.X + 2, p.Y}] = '-'
	box[[2]int{p.X + 3, p.Y}] = '-'
	box[[2]int{p.X + 4, p.Y}] = '+'
	box[[2]int{p.X, p.Y + 1}] = '|'
	for i, r := range labelRunes {
		box[[2]int{p.X + 1 + i, p.Y + 1}] = r
	}
	box[[2]int{p.X + 4, p.Y + 1}] = '|'
	box[[2]int{p.X, p.Y + 2}] = '+'
	box[[2]int{p.X + 1, p.Y + 2}] = '-'
	box[[2]int{p.X + 2, p.Y + 2}] = '-'
	box[[2]int{p.X + 3, p.Y + 2}] = '-'
	box[[2]int{p.X + 4, p.Y + 2}] = '+'

	if m.table.GridType == table.GridTypeHex {
		switch p.Facing % 6 {
		case 0:
			box[[2]int{p.X + 2, p.Y}] = '^'
		case 1:
			box[[2]int{p.X + 4, p.Y}] = '/'
		case 2:
			box[[2]int{p.X + 4, p.Y + 2}] = '\\'
		case 3:
			box[[2]int{p.X + 2, p.Y + 2}] = 'v'
		case 4:
			box[[2]int{p.X, p.Y + 2}] = '/'
		case 5:
			box[[2]int{p.X, p.Y}] = '\\'
		}
	} else {
		switch p.Facing % 4 {
		case 0:
			box[[2]int{p.X + 2, p.Y}] = '^'
		case 1:
			box[[2]int{p.X + 4, p.Y + 1}] = '>'
		case 2:
			box[[2]int{p.X + 2, p.Y + 2}] = 'v'
		case 3:
			box[[2]int{p.X, p.Y + 1}] = '<'
		}
	}

	return box
}

func (m TableViewModel) topBar() string {
	if m.mode == modeSaveName {
		left := styles.Highlight.Render("Save as: ") + m.nameInput.View()
		right := styles.Subtle.Render("enter: save  esc: cancel")
		leftWidth := lipgloss.Width(left)
		rightWidth := lipgloss.Width(right)
		gap := m.width - leftWidth - rightWidth
		if gap < 1 {
			gap = 1
		}
		return left + strings.Repeat(" ", gap) + right
	}

	var left string
	if m.table.GridType != table.GridTypeNone {
		wx := m.cursorX + m.panX
		wy := m.cursorY + m.panY
		if col, row, ok := m.detectCell(wx, wy); ok {
			left = fmt.Sprintf("Cell: %d,%d", col, row)
		} else {
			left = fmt.Sprintf("Pos: %d,%d", wx, wy)
		}
		left += fmt.Sprintf("  L:%d", m.currentLayer)
		if m.panMode {
			left += "  " + styles.Highlight.Render("PAN")
		}
		switch m.mode {
		case modeMPending:
			left += "  " + styles.Highlight.Render("m-")
		case modeDrawMenu:
			left += "  " + styles.Highlight.Render("DRAW: l=line b=box")
		case modeDrawLine:
			left += "  " + styles.Highlight.Render("LINE")
		case modeDrawBox:
			left += "  " + styles.Highlight.Render(fmt.Sprintf("BOX %dx%d", m.boxW, m.boxH))
		case modeText:
			left += "  " + styles.Highlight.Render("TEXT")
		case modeMove:
			left += "  " + styles.Highlight.Render("MOVE")
		case modeLayer:
			left += "  " + styles.Highlight.Render("LAYER")
		case modeTokenMenu:
			left += "  " + styles.Highlight.Render("TOKENS")
		case modeTokenEdit:
			left += "  " + styles.Highlight.Render("TOKEN EDIT")
		case modeTokenColor:
			left += "  " + styles.Highlight.Render("COLOR")
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
	case modeMPending:
		right = styles.Subtle.Render("m: move  d: draw  t: text  l: layer  esc: cancel")
	case modeLayer:
		if m.prevMode == modeMove {
			right = styles.Subtle.Render("+/-: layer  enter: place  esc/l: back")
		} else {
			right = styles.Subtle.Render("+/-: layer  esc/l: back")
		}
	case modeDrawMenu:
		right = styles.Subtle.Render("l: line  b: box  esc: cancel")
	case modeDrawLine:
		right = styles.Subtle.Render("hjkl: draw  ctrl+s: commit  esc: cancel")
	case modeDrawBox:
		right = styles.Subtle.Render("hjkl: resize  ctrl+s: commit  esc: cancel")
	case modeText:
		right = styles.Subtle.Render("type text  enter: newline  arrows: move  ctrl+s: commit  esc: cancel")
	case modeMove:
		right = styles.Subtle.Render("hjkl: move  enter: place  m: mode  esc: cancel")
	case modeTokenMenu:
		if m.tokenCreateMode {
			right = styles.Subtle.Render("type name  enter: create  esc: cancel")
		} else if m.tokenFolderMode {
			right = styles.Subtle.Render("type folder name  enter: create  esc: cancel")
		} else {
			right = styles.Subtle.Render("j/k: navigate  enter: place  n: new  e: edit  d: delete  f: folder  esc: close")
		}
	case modeTokenEdit:
		right = styles.Subtle.Render("type Key: Value  enter: newline  ctrl+s: save  esc: cancel")
	case modeTokenColor:
		right = styles.Subtle.Render("j/k: navigate  enter: select  esc: cancel")
	default:
		right = styles.Subtle.Render("hjkl: move  tab: next  z: pan  m: mode  T: tokens  r: rotate  c: color  q: quit  s: save")
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := m.width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

func (m TableViewModel) viewQuitDialog() string {
	s := fmt.Sprintf("\n\n%s\n\n", styles.Title.Render("Quit?"))
	s += "  Are you sure you want to quit?\n"
	s += "  Unsaved changes will be lost.\n\n"
	s += styles.Subtle.Render("  y/enter: yes  n/esc: no")
	return s
}

func (m TableViewModel) viewDeleteTokenDialog() string {
	name := "this token"
	if m.deleteTokenIdx >= 0 && m.deleteTokenIdx < len(m.table.TokenPlacements) {
		p := m.table.TokenPlacements[m.deleteTokenIdx]
		if def := m.tokenLib.FindTokenDef(p.TokenID); def != nil && len(def.Properties) > 0 && def.Properties[0].Value != "" {
			name = def.Properties[0].Value
		}
	}
	s := fmt.Sprintf("\n\n%s\n\n", styles.Title.Render("Delete Token?"))
	s += fmt.Sprintf("  Remove %s from the table?\n\n", name)
	s += styles.Subtle.Render("  y/enter: yes  n/esc: no")
	return s
}

func (m TableViewModel) viewTokenColor() string {
	var sb strings.Builder
	sb.WriteString(m.topBar())
	sb.WriteByte('\n')
	title := "Token Color"
	if m.colorOverlayPositions != nil {
		title = "Color"
	}
	sb.WriteString(fmt.Sprintf("\n  %s\n\n", styles.Title.Render(title)))
	for i, tc := range tokenColors {
		prefix := "  "
		if i == m.colorCursor {
			prefix = "> "
		}
		swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(tc.color)).Render("██")
		sb.WriteString(fmt.Sprintf("%s%s %s\n", prefix, swatch, tc.name))
	}
	return sb.String()
}

func (m TableViewModel) viewTokenMenu() string {
	var sb strings.Builder
	sb.WriteString(m.topBar())
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf("\n  %s\n\n", styles.Title.Render("Token Library")))

	if m.tokenConfirmDelete && m.tokenMenuCursor >= 0 && m.tokenMenuCursor < len(m.tokenMenuItems) {
		item := m.tokenMenuItems[m.tokenMenuCursor]
		name := "this item"
		if item.kind == tmToken && item.tokenIdx < len(m.tokenLib.Defs) {
			if len(m.tokenLib.Defs[item.tokenIdx].Properties) > 0 {
				name = m.tokenLib.Defs[item.tokenIdx].Properties[0].Value
			}
		} else if item.kind == tmFolder {
			name = "folder [" + item.folder + "]"
		}
		sb.WriteString(fmt.Sprintf("  Delete %s?\n\n", name))
		sb.WriteString(styles.Subtle.Render("  y/enter: yes  n/esc: no"))
		return sb.String()
	}

	if m.tokenCreateMode {
		sb.WriteString("  New token: ")
		sb.WriteString(m.tokenNameInput.View())
		sb.WriteByte('\n')
		return sb.String()
	}
	if m.tokenFolderMode {
		sb.WriteString("  New folder: ")
		sb.WriteString(m.tokenNameInput.View())
		sb.WriteByte('\n')
		return sb.String()
	}

	if len(m.tokenMenuItems) == 0 {
		sb.WriteString("  No tokens yet. Press 'n' to create one.\n")
		return sb.String()
	}

	for i, item := range m.tokenMenuItems {
		prefix := "  "
		if i == m.tokenMenuCursor {
			prefix = "> "
		}
		if item.kind == tmFolder {
			arrow := "> "
			if m.expandedFolders[item.folder] {
				arrow = "v "
			}
			sb.WriteString(prefix + styles.Highlight.Render(arrow+"["+item.folder+"]") + "\n")
		} else {
			td := m.tokenLib.Defs[item.tokenIdx]
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

func (m TableViewModel) viewTokenEdit() string {
	var sb strings.Builder
	sb.WriteString(m.topBar())
	sb.WriteByte('\n')

	if m.tokenEditIdx >= 0 && m.tokenEditIdx < len(m.tokenLib.Defs) {
		td := m.tokenLib.Defs[m.tokenEditIdx]
		name := "Token"
		if len(td.Properties) > 0 && td.Properties[0].Value != "" {
			name = td.Properties[0].Value
		}
		sb.WriteString(fmt.Sprintf("\n  %s\n\n", styles.Title.Render("Edit: "+name)))
	} else {
		sb.WriteString(fmt.Sprintf("\n  %s\n\n", styles.Title.Render("Edit Token")))
	}

	sb.WriteString(m.tokenEditor.View())
	return sb.String()
}
