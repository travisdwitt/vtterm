package views

import (
	"strings"
	"unicode/utf8"
)

type menuItem struct {
	label string
	keys  []string
}

type ctxMenu struct {
	items []menuItem
	x, y  int
	idx   int
}

func (c *ctxMenu) size() (int, int) {
	w := 0
	for _, it := range c.items {
		w = max(w, utf8.RuneCountInString(it.label))
	}
	return w + 4, len(c.items) + 2
}

func (c *ctxMenu) hit(x, y int) (int, bool) {
	w, h := c.size()
	if x < c.x || x >= c.x+w || y < c.y || y >= c.y+h {
		return -1, false
	}
	i := y - c.y - 1
	if i < 0 || i >= len(c.items) || x == c.x || x == c.x+w-1 {
		i = -1
	}
	return i, true
}

func (c *ctxMenu) cell(x, y int) (r rune, in, sel bool) {
	if c == nil {
		return 0, false, false
	}
	w, h := c.size()
	col, row := x-c.x, y-c.y
	if col < 0 || col >= w || row < 0 || row >= h {
		return 0, false, false
	}
	edge := col == 0 || col == w-1
	switch {
	case row == 0 && col == 0:
		return '┌', true, false
	case row == 0 && col == w-1:
		return '┐', true, false
	case row == h-1 && col == 0:
		return '└', true, false
	case row == h-1 && col == w-1:
		return '┘', true, false
	case row == 0 || row == h-1:
		return '─', true, false
	case edge:
		return '│', true, false
	}
	label := []rune(" " + c.items[row-1].label + strings.Repeat(" ", w))
	return label[col-1], true, row-1 == c.idx
}
