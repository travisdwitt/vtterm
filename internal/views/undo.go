package views

import (
	"cmp"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/traviswitt/vtterm/internal/styles"
	"github.com/traviswitt/vtterm/internal/table"
)

const maxUndo = 100

type snap struct {
	overlay []table.OverlayChar
	places  []table.TokenPlacement
}

func (m *TableViewModel) snapshot() snap {
	m.syncOverlayToTable()
	ol := slices.Clone(m.table.Overlay)
	slices.SortFunc(ol, func(a, b table.OverlayChar) int {
		return cmp.Or(cmp.Compare(a.Layer, b.Layer), cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	return snap{ol, slices.Clone(m.table.TokenPlacements)}
}

func (m *TableViewModel) restore(s snap) {
	m.table.Overlay = slices.Clone(s.overlay)
	m.table.TokenPlacements = slices.Clone(s.places)
	m.layers = make(map[int]map[[2]int]rune)
	m.layerColors = make(map[int]map[[2]int]string)
	m.layerGroups = make(map[int]map[[2]int]string)
	m.layerLocks = make(map[int]map[[2]int]bool)
	m.loadOverlayFromTable()
	m.inspectTokenIdx = -1
	m.base = s
}

func (m *TableViewModel) stable() bool {
	return m.mode == modeNormal && m.moving == nil && m.movingTokenIdx < 0 && m.pending == nil
}

func (m *TableViewModel) record() {
	if !m.stable() {
		return
	}
	cur := m.snapshot()
	if slices.Equal(cur.overlay, m.base.overlay) && slices.Equal(cur.places, m.base.places) {
		return
	}
	m.undo = append(m.undo, m.base)
	if len(m.undo) > maxUndo {
		m.undo = m.undo[1:]
	}
	m.redo = nil
	m.base = cur
}

func (m *TableViewModel) step(from, to *[]snap, name string) tea.Cmd {
	if len(*from) == 0 {
		m.statusMsg = styles.Subtle.Render("Nothing to " + name)
		return clearAfter(2 * time.Second)
	}
	*to = append(*to, m.base)
	if len(*to) > maxUndo {
		*to = (*to)[1:]
	}
	n := len(*from) - 1
	s := (*from)[n]
	*from = (*from)[:n]
	m.restore(s)
	m.clampCursor()
	m.statusMsg = styles.Status.Render(name)
	return clearAfter(2 * time.Second)
}
