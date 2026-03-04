package table

import (
	"crypto/rand"
	"fmt"
	"time"
)

type GridType string

const (
	GridTypeGrid GridType = "grid"
	GridTypeHex  GridType = "hex"
	GridTypeNone GridType = "none"
)

type OverlayChar struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	R     string `json:"r"`
	Layer int    `json:"layer,omitempty"`
	Color string `json:"color,omitempty"`
}

type TokenProperty struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type TokenDef struct {
	ID         string          `json:"id"`
	Folder     string          `json:"folder,omitempty"`
	Properties []TokenProperty `json:"properties"`
}

type TokenPlacement struct {
	TokenID string `json:"token_id"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Layer   int    `json:"layer,omitempty"`
	Facing  int    `json:"facing,omitempty"` // 0=up, rotates clockwise
	Color   string `json:"color,omitempty"`  // ANSI color number, empty = default
}

type TokenLibrary struct {
	Defs    []TokenDef `json:"defs,omitempty"`
	Folders []string   `json:"folders,omitempty"`
}

type Table struct {
	Name            string           `json:"name"`
	GridType        GridType         `json:"grid_type"`
	Width           int              `json:"width"`
	Height          int              `json:"height"`
	Overlay         []OverlayChar    `json:"overlay,omitempty"`
	TokenPlacements []TokenPlacement `json:"token_placements,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

func NewTokenID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (lib *TokenLibrary) FindTokenDef(id string) *TokenDef {
	for i := range lib.Defs {
		if lib.Defs[i].ID == id {
			return &lib.Defs[i]
		}
	}
	return nil
}

func (td *TokenDef) IsDisabled() bool {
	for _, p := range td.Properties {
		if p.Key == "Disabled" {
			return true
		}
	}
	return false
}

func (td *TokenDef) DisplayLabel() string {
	if len(td.Properties) == 0 || td.Properties[0].Value == "" {
		return "???"
	}
	runes := []rune(td.Properties[0].Value)
	if len(runes) > 3 {
		runes = runes[:3]
	}
	return fmt.Sprintf("%-3s", string(runes))
}
