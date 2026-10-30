package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/traviswitt/vtterm/internal/table"
)

func send(m tea.Model, msgs ...tea.Msg) TableViewModel {
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}
	return m.(TableViewModel)
}

func click(x, y int, b tea.MouseButton) tea.Msg {
	return tea.MouseClickMsg{X: x, Y: y, Button: b}
}

func TestTableMouse(t *testing.T) {
	lib := &table.TokenLibrary{Defs: []table.TokenDef{{ID: "a", Properties: []table.TokenProperty{{Key: "Name", Value: "Gob"}}}}}
	tb := table.Table{Name: "t", GridType: table.GridTypeGrid, Width: 5, Height: 5,
		TokenPlacements: []table.TokenPlacement{{TokenID: "a", X: 10, Y: 5}}}
	m := NewTableView(tb, lib, 120, 40)

	m = send(m, click(11, 7, tea.MouseLeft), tea.MouseReleaseMsg{X: 11, Y: 7, Button: tea.MouseLeft})
	if p := m.table.TokenPlacements[0]; p.X != 10 || p.Y != 5 || m.mode != modeNormal {
		t.Fatalf("plain click moved token: %+v mode %v", p, m.mode)
	}

	m = send(m, click(11, 7, tea.MouseLeft),
		tea.MouseMotionMsg{X: 31, Y: 7, Button: tea.MouseLeft},
		tea.MouseReleaseMsg{X: 31, Y: 7, Button: tea.MouseLeft})
	if p := m.table.TokenPlacements[0]; p.X == 10 || m.mode != modeNormal {
		t.Fatalf("drag did not move token: %+v mode %v", p, m.mode)
	}

	m = send(m, click(60, 30, tea.MouseRight))
	if m.menu == nil {
		t.Fatal("right click did not open menu")
	}
	box := -1
	for i, it := range m.menu.items {
		if it.label == "Box" {
			box = i
		}
	}
	m = send(m, click(m.menu.x+2, m.menu.y+1+box, tea.MouseLeft))
	if m.tool != "b" || m.mode != modeNormal || m.menu != nil {
		t.Fatalf("menu Box: tool %q mode %v menu %v", m.tool, m.mode, m.menu)
	}
	m = send(m, tea.MouseMotionMsg{X: 62, Y: 31})
	if len(m.activeOverlay()) != 0 || m.mode != modeNormal {
		t.Fatal("drew before button press")
	}
	m = send(m, click(60, 30, tea.MouseLeft), tea.MouseMotionMsg{X: 66, Y: 33, Button: tea.MouseLeft})
	if m.mode != modeDrawBox || len(m.activeOverlay()) != 0 {
		t.Fatalf("not drawing while held: mode %v", m.mode)
	}
	m = send(m, tea.MouseReleaseMsg{X: 66, Y: 33, Button: tea.MouseLeft})
	if m.mode != modeNormal || len(m.activeOverlay()) == 0 {
		t.Fatalf("box not committed: mode %v overlay %d", m.mode, len(m.activeOverlay()))
	}

	m.tool = "c"
	n := len(m.activeOverlay())
	m = send(m, click(40, 20, tea.MouseLeft), tea.MouseMotionMsg{X: 48, Y: 20, Button: tea.MouseLeft},
		tea.MouseReleaseMsg{X: 48, Y: 20, Button: tea.MouseLeft})
	if m.mode != modeNormal || len(m.activeOverlay()) <= n {
		t.Fatalf("circle not committed: mode %v", m.mode)
	}
	m.tool = "b"
	n = len(m.activeOverlay())
	m = send(m, click(10, 20, tea.MouseLeft), tea.MouseReleaseMsg{X: 10, Y: 20, Button: tea.MouseLeft})
	if m.mode != modeNormal || len(m.activeOverlay()) != n {
		t.Fatal("click without drag created a shape")
	}

	m = send(m, click(120, 5, tea.MouseWheelDown))
	_ = m.View()
}

func TestLockUnlock(t *testing.T) {
	lib := &table.TokenLibrary{Defs: []table.TokenDef{{ID: "a", Properties: []table.TokenProperty{{Key: "Name", Value: "Gob"}}}}}
	tb := table.Table{Name: "t", GridType: table.GridTypeGrid, Width: 5, Height: 5,
		TokenPlacements: []table.TokenPlacement{{TokenID: "a", X: 10, Y: 5}}}
	m := NewTableView(tb, lib, 120, 40)
	drag := func() {
		m = send(m, click(11, 7, tea.MouseLeft),
			tea.MouseMotionMsg{X: 31, Y: 7, Button: tea.MouseLeft},
			tea.MouseReleaseMsg{X: 31, Y: 7, Button: tea.MouseLeft})
	}
	pick := func(label string) {
		m = send(m, click(11, 7, tea.MouseRight))
		for i, it := range m.menu.items {
			if it.label == label {
				m = send(m, click(m.menu.x+2, m.menu.y+1+i, tea.MouseLeft))
				return
			}
		}
		t.Fatalf("no %q item", label)
	}

	pick("Lock")
	drag()
	if m.table.TokenPlacements[0].X != 10 || m.mode != modeNormal {
		t.Fatalf("locked token moved: %+v", m.table.TokenPlacements[0])
	}
	m.panX, m.panY = 0, 0
	m = send(m, tea.KeyPressMsg{Code: 'm', Text: "m"}, tea.KeyPressMsg{Code: 'm', Text: "m"})
	if m.mode != modeNormal {
		t.Fatal("keyboard move allowed on locked token")
	}

	pick("Unlock")
	drag()
	if m.table.TokenPlacements[0].X == 10 {
		t.Fatal("unlocked token did not move")
	}
}

func TestHoverTip(t *testing.T) {
	lib := &table.TokenLibrary{Defs: []table.TokenDef{{ID: "a", Properties: []table.TokenProperty{{Key: "Name", Value: "Gob"}, {Key: "HP", Value: "7"}}}}}
	tb := table.Table{Name: "t", GridType: table.GridTypeGrid, Width: 5, Height: 5,
		TokenPlacements: []table.TokenPlacement{{TokenID: "a", X: 10, Y: 5}}}
	m := NewTableView(tb, lib, 120, 40)
	m = send(m, tea.MouseMotionMsg{X: 11, Y: 7})
	if m.tip != nil {
		t.Fatal("tooltip shown without delay")
	}
	m = send(m, hoverMsg{seq: m.hoverSeq - 1})
	if m.tip != nil {
		t.Fatal("stale timer showed tooltip")
	}
	m = send(m, hoverMsg{seq: m.hoverSeq})
	if m.tip == nil || len(m.tip.items) != 2 {
		t.Fatalf("no tooltip on hover: %+v", m.tip)
	}
	m = send(m, tea.MouseMotionMsg{X: 60, Y: 30})
	if m.tip != nil {
		t.Fatal("tooltip stayed after leaving token")
	}
}

func TestTokenLibSaved(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	lib := &table.TokenLibrary{}
	tb := table.Table{Name: "t", GridType: table.GridTypeGrid, Width: 5, Height: 5}
	var m tea.Model = NewTableView(tb, lib, 120, 40)
	var cmd tea.Cmd
	for _, k := range []tea.KeyPressMsg{{Code: 'T', Text: "T"}, {Code: 'n', Text: "n"}, {Code: 'o', Text: "o"}, {Code: tea.KeyEnter}} {
		m, cmd = m.Update(k)
	}
	if cmd == nil {
		t.Fatal("creating a token returned no save command")
	}
	cmd()
	got, err := table.LoadTokenLibrary()
	if err != nil || len(got.Defs) != 1 {
		t.Fatalf("token not persisted: %+v %v", got, err)
	}
}

func TestHiddenProps(t *testing.T) {
	lib := &table.TokenLibrary{Defs: []table.TokenDef{{ID: "a", Properties: []table.TokenProperty{{Key: "Name", Value: "Bob"}, {Key: "H-Secret", Value: "x"}}}}}
	tb := table.Table{Name: "t", GridType: table.GridTypeGrid, Width: 5, Height: 5,
		TokenPlacements: []table.TokenPlacement{{TokenID: "a", X: 10, Y: 5}}}
	m := NewTableView(tb, lib, 120, 40)
	m = send(m, tea.MouseMotionMsg{X: 11, Y: 7})
	m = send(m, hoverMsg{seq: m.hoverSeq})
	if m.tip == nil || len(m.tip.items) != 1 || m.tip.items[0].label != "Name: Bob" {
		t.Fatalf("hidden property shown: %+v", m.tip)
	}
}

func TestUndoRedo(t *testing.T) {
	lib := &table.TokenLibrary{Defs: []table.TokenDef{{ID: "a", Properties: []table.TokenProperty{{Key: "Name", Value: "Gob"}}}}}
	tb := table.Table{Name: "t", GridType: table.GridTypeGrid, Width: 5, Height: 5,
		TokenPlacements: []table.TokenPlacement{{TokenID: "a", X: 10, Y: 5}}}
	m := NewTableView(tb, lib, 120, 40)
	key := func(k tea.KeyPressMsg) { m = send(m, k) }

	m = send(m, click(11, 7, tea.MouseLeft),
		tea.MouseMotionMsg{X: 31, Y: 7, Button: tea.MouseLeft},
		tea.MouseReleaseMsg{X: 31, Y: 7, Button: tea.MouseLeft})
	moved := m.table.TokenPlacements[0].X
	if moved == 10 || len(m.undo) != 1 {
		t.Fatalf("move not recorded: x=%d undo=%d", moved, len(m.undo))
	}

	m.tool = "b"
	m = send(m, click(60, 30, tea.MouseLeft), tea.MouseMotionMsg{X: 66, Y: 33, Button: tea.MouseLeft},
		tea.MouseReleaseMsg{X: 66, Y: 33, Button: tea.MouseLeft})
	if len(m.activeOverlay()) == 0 || len(m.undo) != 2 {
		t.Fatalf("box not recorded: undo=%d", len(m.undo))
	}

	key(tea.KeyPressMsg{Code: 'U', Text: "U"})
	if len(m.activeOverlay()) != 0 || m.table.TokenPlacements[0].X != moved {
		t.Fatal("undo did not remove box")
	}
	key(tea.KeyPressMsg{Code: 'U', Text: "U"})
	if m.table.TokenPlacements[0].X != 10 {
		t.Fatal("undo did not restore token")
	}
	key(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	key(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if len(m.activeOverlay()) == 0 || m.table.TokenPlacements[0].X != moved || len(m.redo) != 0 {
		t.Fatal("redo did not reapply")
	}

	for i := 0; i < 120; i++ {
		m.table.TokenPlacements[0].Facing = i + 1
		m.record()
	}
	if len(m.undo) != maxUndo {
		t.Fatalf("undo not capped: %d", len(m.undo))
	}
}

func TestInspectTip(t *testing.T) {
	lib := &table.TokenLibrary{Defs: []table.TokenDef{{ID: "a", Properties: []table.TokenProperty{{Key: "Name", Value: "Bob"}, {Key: "H-Secret", Value: "x"}}}}}
	tb := table.Table{Name: "t", GridType: table.GridTypeGrid, Width: 5, Height: 5,
		TokenPlacements: []table.TokenPlacement{{TokenID: "a", X: 10, Y: 5}}}
	m := NewTableView(tb, lib, 120, 40)
	m = send(m, click(11, 7, tea.MouseLeft), tea.MouseReleaseMsg{X: 11, Y: 7, Button: tea.MouseLeft},
		tea.KeyPressMsg{Code: 'i', Text: "i"})
	c := m.overlay()
	if c == nil || len(c.items) != 1 || c.items[0].label != "Name: Bob" {
		t.Fatalf("inspect tooltip wrong: %+v", c)
	}
	if v := m.View().Content; !strings.Contains(v, "┌") || !strings.Contains(v, "Name: Bob") {
		t.Fatal("inspect tooltip not rendered")
	}
	m = send(m, tea.KeyPressMsg{Code: 'i', Text: "i"})
	if m.overlay() != nil {
		t.Fatal("second i did not close tooltip")
	}
}
