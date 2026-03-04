package views

import (
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/traviswitt/vtterm/internal/table"
)

type tokenMenuItemKind int

const (
	tmFolder tokenMenuItemKind = iota
	tmToken
)

type tokenMenuItem struct {
	kind     tokenMenuItemKind
	folder   string
	tokenIdx int
}

type clearStatusMsg struct{}

func buildTokenMenu(lib *table.TokenLibrary, expanded map[string]bool) []tokenMenuItem {
	var items []tokenMenuItem
	for i, td := range lib.Defs {
		if td.Folder == "" {
			items = append(items, tokenMenuItem{kind: tmToken, tokenIdx: i})
		}
	}
	folders := make([]string, len(lib.Folders))
	copy(folders, lib.Folders)
	sort.Strings(folders)
	for _, f := range folders {
		items = append(items, tokenMenuItem{kind: tmFolder, folder: f})
		if expanded[f] {
			for i, td := range lib.Defs {
				if td.Folder == f {
					items = append(items, tokenMenuItem{kind: tmToken, folder: f, tokenIdx: i})
				}
			}
		}
	}
	return items
}

func deleteTokenDef(lib *table.TokenLibrary, idx int) {
	lib.Defs = append(lib.Defs[:idx], lib.Defs[idx+1:]...)
}

func deleteFolder(lib *table.TokenLibrary, name string) {
	for i := range lib.Defs {
		if lib.Defs[i].Folder == name {
			lib.Defs[i].Folder = ""
		}
	}
	folders := lib.Folders[:0]
	for _, f := range lib.Folders {
		if f != name {
			folders = append(folders, f)
		}
	}
	lib.Folders = folders
}

func clearAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}
