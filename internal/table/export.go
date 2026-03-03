package table

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/traviswitt/vtterm/internal/grid"
)

func Export(t *Table) (string, error) {
	var rendered string
	switch t.GridType {
	case GridTypeGrid:
		rendered = grid.RenderSquare(t.Width, t.Height)
	case GridTypeHex:
		if t.HexOrientation == HexFlatTop {
			rendered = grid.RenderFlatHex(t.Width, t.Height)
		} else {
			rendered = grid.RenderPointyHex(t.Width, t.Height)
		}
	case GridTypeNone:
		rendered = "(blank canvas)\n"
	}

	dir := SaveDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating export directory: %w", err)
	}

	filename := sanitize(t.Name) + ".txt"
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
