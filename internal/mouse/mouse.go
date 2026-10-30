package mouse

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/traviswitt/vtterm/internal/styles"
)

type Kind int

const (
	Press Kind = iota
	Release
	Motion
	Wheel
)

type Event struct {
	X, Y int
	Btn  tea.MouseButton
	Kind Kind
}

func Of(m tea.MouseMsg) Event {
	mo := m.Mouse()
	e := Event{X: mo.X, Y: mo.Y, Btn: mo.Button, Kind: Wheel}
	switch m.(type) {
	case tea.MouseClickMsg:
		e.Kind = Press
	case tea.MouseReleaseMsg:
		e.Kind = Release
	case tea.MouseMotionMsg:
		e.Kind = Motion
	}
	return e
}

func (e Event) Left() bool  { return e.Kind == Press && e.Btn == tea.MouseLeft }
func (e Event) Right() bool { return e.Kind == Press && e.Btn == tea.MouseRight }

func Key(s string) tea.KeyPressMsg {
	switch s {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	}
	if k, ok := strings.CutPrefix(s, "ctrl+"); ok {
		return tea.KeyPressMsg{Code: []rune(k)[0], Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

func Feed(m tea.Model, keys ...string) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	for _, k := range keys {
		var c tea.Cmd
		m, c = m.Update(Key(k))
		cmds = append(cmds, c)
	}
	return m, tea.Batch(cmds...)
}

// ListKeys maps a click, wheel or right-click on a vertical list whose first
// item is on screen row first. Clicking the current item (or any item when
// act is set) also sends enter; right-click sends esc.
func ListKeys(e Event, first, n, cur int, act bool) []string {
	switch {
	case e.Kind == Wheel && e.Btn == tea.MouseWheelUp:
		return []string{"up"}
	case e.Kind == Wheel && e.Btn == tea.MouseWheelDown:
		return []string{"down"}
	case e.Right():
		return []string{"esc"}
	case e.Left():
		i := e.Y - first
		if i < 0 || i >= n {
			return nil
		}
		var ks []string
		for c := cur; c < i; c++ {
			ks = append(ks, "down")
		}
		for c := cur; c > i; c-- {
			ks = append(ks, "up")
		}
		if act || i == cur {
			ks = append(ks, "enter")
		}
		return ks
	}
	return nil
}

func YesNo(e Event) []string {
	switch {
	case e.Left():
		return []string{"y"}
	case e.Right():
		return []string{"n"}
	}
	return nil
}

type Button struct{ Label, Key string }

func Bar(bs []Button) string {
	parts := make([]string, len(bs))
	for i, b := range bs {
		parts[i] = "[" + b.Label + "]"
	}
	return "  " + styles.Highlight.Render(strings.Join(parts, " "))
}

func Hit(bs []Button, e Event, row int) []string {
	if !e.Left() || e.Y != row {
		return nil
	}
	x := 2
	for _, b := range bs {
		w := len(b.Label) + 2
		if e.X >= x && e.X < x+w {
			return []string{b.Key}
		}
		x += w + 1
	}
	return nil
}
