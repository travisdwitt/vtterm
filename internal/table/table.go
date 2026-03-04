package table

import "time"

type GridType string

const (
	GridTypeGrid GridType = "grid"
	GridTypeHex  GridType = "hex"
	GridTypeNone GridType = "none"
)

type Table struct {
	Name      string    `json:"name"`
	GridType  GridType  `json:"grid_type"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
