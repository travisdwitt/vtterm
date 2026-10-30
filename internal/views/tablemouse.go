package views

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/traviswitt/vtterm/internal/mouse"
	"github.com/traviswitt/vtterm/internal/table"
)

type dragKind int

const (
	dragNone dragKind = iota
	dragPan
	dragMove
	dragDraw
)

func (m TableViewModel) updateMouse(message tea.MouseMsg) (tea.Model, tea.Cmd) {
	e := mouse.Of(message)
	m.tip = nil
	m.hoverSeq++
	if m.showQuit || m.deleteTokenIdx >= 0 || m.removeFillGroup != "" || m.confirmDeleteShape {
		return mouse.Feed(m, mouse.YesNo(e)...)
	}
	if m.table.GridType == table.GridTypeNone {
		return m, nil
	}
	if m.menu != nil {
		return m.menuMouse(e)
	}
	switch m.mode {
	case modeNormal:
		return m.normalMouse(e)
	case modeMove:
		return m.moveMouse(e)
	case modeDrawBox, modeDrawLine, modeDrawCircle, modeText:
		return m.drawMouse(e)
	case modeLayer:
		k := ""
		switch {
		case e.Kind == mouse.Wheel && e.Btn == tea.MouseWheelUp:
			k = "+"
		case e.Kind == mouse.Wheel && e.Btn == tea.MouseWheelDown:
			k = "-"
		case e.Left():
			k = "enter"
		case e.Right():
			k = "esc"
		}
		if k != "" {
			return mouse.Feed(m, k)
		}
	case modeTokenMenu:
		if m.tokenConfirmDelete {
			return mouse.Feed(m, mouse.YesNo(e)...)
		}
		if m.tokenCreateMode || m.tokenFolderMode {
			return mouse.Feed(m, mouse.ListKeys(e, 0, 0, 0, false)...)
		}
		if ks := mouse.Hit(tokenBtns, e, 3); ks != nil {
			return mouse.Feed(m, ks...)
		}
		return mouse.Feed(m, tokenListKeys(e, m.tokenMenuItems, m.tokenMenuCursor)...)
	case modeTokenEdit:
		switch {
		case e.Left():
			m.tokenEditor.Click(e.Y-4, e.X-2)
		case e.Right():
			return mouse.Feed(m, "ctrl+s")
		}
	case modeTokenColor:
		first := 4
		if m.fillPositions != nil {
			first = 7
			if e.Left() && e.Y == 4 {
				m.fillOpacity = (m.fillOpacity + 1) % len(fillRunes)
				return m, nil
			}
		}
		return mouse.Feed(m, mouse.ListKeys(e, first, len(tokenColors), m.colorCursor, true)...)
	default:
		return mouse.Feed(m, mouse.ListKeys(e, 0, 0, 0, false)...)
	}
	return m, nil
}

func (m *TableViewModel) pointCursor(e mouse.Event) (int, int) {
	m.cursorX, m.cursorY = e.X, e.Y-1
	m.clampCursor()
	return e.X + m.panX, e.Y - 1 + m.panY
}

func (m TableViewModel) normalMouse(e mouse.Event) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch {
	case e.Kind == mouse.Wheel:
		switch e.Btn {
		case tea.MouseWheelUp:
			m.panY -= 2
		case tea.MouseWheelDown:
			m.panY += 2
		case tea.MouseWheelLeft:
			m.panX -= 2
		case tea.MouseWheelRight:
			m.panX += 2
		}
	case e.Kind == mouse.Motion && m.drag == dragPan:
		m.panX -= e.X - m.lastX
		m.panY -= e.Y - m.lastY
		m.lastX, m.lastY = e.X, e.Y
	case e.Kind == mouse.Release:
		m.drag = dragNone
	case e.Kind == mouse.Motion:
		cmd = m.hover(e)
	case e.Left() && m.tool != "":
		k := m.tool
		m.tool = ""
		m.pointCursor(e)
		m.drag = dragDraw
		return mouse.Feed(m, "m", "d", k)
	case e.Left():
		wx, wy := m.pointCursor(e)
		m.grabX, m.grabY = wx, wy
		m.moved = false
		locked := m.lockedAt(wx, wy)
		if ti := m.findTokenPlacementAt(wx, wy); ti >= 0 && !locked {
			m.beginTokenMove(ti)
			m.drag = dragMove
		} else if sel := m.groupSelectOverlay(wx, wy); !locked && len(sel) > 0 {
			m.beginShapeMove(sel)
			m.drag = dragMove
		} else {
			m.drag = dragPan
			m.lastX, m.lastY = e.X, e.Y
		}
	case e.Right() && m.tool != "":
		m.tool = ""
	case e.Right():
		wx, wy := m.pointCursor(e)
		m.grabX, m.grabY = wx, wy
		c := &ctxMenu{items: m.buildMenu()}
		w, h := c.size()
		c.x = max(min(e.X, m.width-w), 0)
		c.y = max(min(e.Y, m.height-h), 1)
		m.menu = c
	}
	return m, cmd
}

func (m TableViewModel) buildMenu() []menuItem {
	wx, wy := m.cursorX+m.panX, m.cursorY+m.panY
	var items []menuItem
	lock := "Lock"
	if m.lockedAt(wx, wy) {
		lock = "Unlock"
	}
	if pi := m.findTokenPlacementAt(wx, wy); pi >= 0 {
		toggle := "Disable"
		if def := m.tokenLib.FindTokenDef(m.table.TokenPlacements[pi].TokenID); def != nil && def.IsDisabled() {
			toggle = "Enable"
		}
		items = []menuItem{
			{"Move", []string{"m", "m"}},
			{lock, []string{"x"}},
			{"Inspect", []string{"i"}},
			{"Edit", []string{"e"}},
			{"Rotate", []string{"r"}},
			{"Color", []string{"c"}},
			{toggle, []string{"d"}},
			{"Delete", []string{"D"}},
		}
	} else if r, ok := m.layers[m.currentLayer][[2]int{wx, wy}]; ok {
		items = []menuItem{
			{"Move", []string{"m", "m"}},
			{lock, []string{"x"}},
			{"Color", []string{"c"}},
			{"Delete", []string{"m", "m", "D"}},
		}
		if isFillRune(r) {
			items = append(items, menuItem{"Remove Fill", []string{"f"}})
		}
	} else {
		items = []menuItem{{"Fill", []string{"f"}}}
	}
	return append(items,
		menuItem{"Line", []string{"tool:l"}},
		menuItem{"Box", []string{"tool:b"}},
		menuItem{"Circle", []string{"tool:c"}},
		menuItem{"Text", []string{"m", "t"}},
		menuItem{"Tokens", []string{"T"}},
		menuItem{"Layer +", []string{"m", "l", "+", "esc"}},
		menuItem{"Layer -", []string{"m", "l", "-", "esc"}},
		menuItem{"Undo", []string{"U"}},
		menuItem{"Redo", []string{"ctrl+r"}},
		menuItem{"Save", []string{"s"}},
		menuItem{"Export", []string{"S"}},
		menuItem{"Quit", []string{"q"}},
	)
}

func (m TableViewModel) menuMouse(e mouse.Event) (tea.Model, tea.Cmd) {
	i, in := m.menu.hit(e.X, e.Y)
	switch {
	case e.Kind == mouse.Motion:
		if i >= 0 {
			m.menu.idx = i
		}
	case e.Left():
		c := m.menu
		m.menu = nil
		if in && i >= 0 {
			if k, ok := strings.CutPrefix(c.items[i].keys[0], "tool:"); ok {
				m.tool = k
				return m, nil
			}
			return mouse.Feed(m, c.items[i].keys...)
		}
	case e.Right():
		m.menu = nil
	}
	return m, nil
}

func (m TableViewModel) moveMouse(e mouse.Event) (tea.Model, tea.Cmd) {
	dragging := m.drag == dragMove
	switch {
	case e.Kind == mouse.Motion:
		dx, dy := e.X+m.panX-m.grabX, e.Y-1+m.panY-m.grabY
		if m.movingTokenIdx >= 0 {
			m.tokenMoveOffsetX, m.tokenMoveOffsetY = dx, dy
		} else {
			m.moveOffsetX, m.moveOffsetY = dx, dy
		}
		m.moved = m.moved || dx != 0 || dy != 0
	case e.Right(), e.Kind == mouse.Release && dragging && !m.moved:
		m.drag = dragNone
		return mouse.Feed(m, "esc")
	case e.Kind == mouse.Release && dragging, e.Left() && !dragging:
		m.drag = dragNone
		if m.movingTokenIdx >= 0 {
			m.snapTokenToCell()
		}
		return mouse.Feed(m, "enter")
	}
	return m, nil
}

func (m TableViewModel) drawMouse(e mouse.Event) (tea.Model, tea.Cmd) {
	switch {
	case e.Left() && m.mode == modeText:
		return mouse.Feed(m, "ctrl+s")
	case e.Right():
		m.drag = dragNone
		return mouse.Feed(m, "esc")
	case e.Kind == mouse.Release && m.drag == dragDraw:
		m.drag = dragNone
		if len(m.pending) <= 1 {
			return mouse.Feed(m, "esc")
		}
		return mouse.Feed(m, "ctrl+s")
	case e.Kind != mouse.Motion:
		return m, nil
	}
	switch m.mode {
	case modeDrawBox:
		wx, wy := m.pointCursor(e)
		m.boxW = max(wx-m.boxAnchorX+1, 1)
		m.boxH = max(wy-m.boxAnchorY+1, 1)
		m.rebuildBoxPending()
	case modeDrawCircle:
		wx, wy := m.pointCursor(e)
		m.circleR = max(abs(wx-m.circleX), 2*abs(wy-m.circleY))
		m.rebuildCirclePending()
	case modeDrawLine:
		tx, ty := e.X, e.Y-1
		for range 400 {
			px, py := m.cursorX, m.cursorY
			switch {
			case px != tx:
				m.depositLineChar(sign(tx-px), 0)
			case py != ty:
				m.depositLineChar(0, sign(ty-py))
			}
			if m.cursorX == px && m.cursorY == py {
				break
			}
		}
	}
	return m, nil
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}

func (m TableViewModel) lockedAt(wx, wy int) bool {
	if pi := m.findTokenPlacementAt(wx, wy); pi >= 0 {
		return m.table.TokenPlacements[pi].Locked
	}
	return m.layerLocks[m.currentLayer][[2]int{wx, wy}]
}

func (m *TableViewModel) toggleLock(wx, wy int) {
	if pi := m.findTokenPlacementAt(wx, wy); pi >= 0 {
		p := &m.table.TokenPlacements[pi]
		p.Locked = !p.Locked
		return
	}
	sel := m.groupSelectOverlay(wx, wy)
	if len(sel) == 0 {
		return
	}
	on := !m.lockedAt(wx, wy)
	if m.layerLocks[m.currentLayer] == nil {
		m.layerLocks[m.currentLayer] = make(map[[2]int]bool)
	}
	for k := range sel {
		if on {
			m.layerLocks[m.currentLayer][k] = true
		} else {
			delete(m.layerLocks[m.currentLayer], k)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (m TableViewModel) overlay() *ctxMenu {
	switch {
	case m.menu != nil:
		return m.menu
	case m.tip != nil:
		return m.tip
	case m.inspectTokenIdx >= 0 && m.inspectTokenIdx < len(m.table.TokenPlacements):
		p := m.table.TokenPlacements[m.inspectTokenIdx]
		return m.tipFor(m.inspectTokenIdx, p.X+6-m.panX, p.Y-m.panY+1)
	}
	return nil
}

type hoverMsg struct{ seq int }

func (m *TableViewModel) hover(e mouse.Event) tea.Cmd {
	if m.tool != "" {
		return nil
	}
	m.hoverX, m.hoverY = e.X, e.Y
	seq := m.hoverSeq
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return hoverMsg{seq} })
}

func (m *TableViewModel) showTip(x, y int) {
	if pi := m.findTokenPlacementAt(x+m.panX, y-1+m.panY); pi >= 0 {
		m.tip = m.tipFor(pi, x+2, y+1)
	}
}

func (m TableViewModel) tipFor(pi, x, y int) *ctxMenu {
	def := m.tokenLib.FindTokenDef(m.table.TokenPlacements[pi].TokenID)
	if def == nil {
		return nil
	}
	c := &ctxMenu{idx: -1}
	for _, p := range def.Visible() {
		c.items = append(c.items, menuItem{label: p.Key + ": " + p.Value})
	}
	if len(c.items) == 0 {
		return nil
	}
	w, h := c.size()
	c.x = max(min(x, m.width-w), 0)
	c.y = max(min(y, m.height-h), 1)
	return c
}
