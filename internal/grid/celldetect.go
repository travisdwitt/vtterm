package grid

import "math"

// Exported geometry constants derived from unexported cellWidth=8, cellHeight=3.
const (
	SquareCellW = 9  // cellWidth + 1 border
	SquareCellH = 4  // cellHeight + 1 border

	FlatHexSpacingX = 10
	FlatHexSpacingY = 6
)

// CenterFunc returns the world-space center of cell (col, row).
type CenterFunc func(col, row int) (x, y int)

// DetectSquareCell returns the cell at world position (wx, wy).
// Returns ok=false if (wx, wy) falls on a border line or is out of bounds.
func DetectSquareCell(wx, wy, cols, rows int) (col, row int, ok bool) {
	if wx < 0 || wy < 0 {
		return 0, 0, false
	}
	if wx%SquareCellW == 0 || wy%SquareCellH == 0 {
		return 0, 0, false
	}
	col = wx / SquareCellW
	row = wy / SquareCellH
	if col >= cols || row >= rows {
		return 0, 0, false
	}
	return col, row, true
}

// SquareCellCenter returns the world-space center of square cell (col, row).
func SquareCellCenter(col, row int) (x, y int) {
	return col*SquareCellW + 4, row*SquareCellH + 2
}

// DetectFlatHexCell returns the flat-top hex cell at world position (wx, wy).
func DetectFlatHexCell(wx, wy, cols, rows int) (col, row int, ok bool) {
	// Estimate column and row from spacing.
	estCol := int(math.Round(float64(wx-7) / float64(FlatHexSpacingX)))
	baseRow := float64(wy-3) / float64(FlatHexSpacingY)
	estRow := int(math.Round(baseRow))

	bestDist := math.MaxFloat64
	bestCol, bestRow := -1, -1

	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			c := estCol + dc
			r := estRow + dr
			if c < 0 || r < 0 || c >= cols || r >= rows {
				continue
			}
			px, py := FlatHexCellCenter(c, r)
			dx := float64(wx - px)
			dy := float64(wy - py)
			d := dx*dx + dy*dy
			if d < bestDist {
				bestDist = d
				bestCol = c
				bestRow = r
			}
		}
	}

	if bestCol < 0 {
		return 0, 0, false
	}

	// Check if point is within hex interior (approximate).
	px, py := FlatHexCellCenter(bestCol, bestRow)
	dx := abs(wx - px)
	dy := abs(wy - py)
	if dx > 4 || dy > 2 {
		return 0, 0, false
	}

	return bestCol, bestRow, true
}

// FlatHexCellCenter returns the world-space center of flat-top hex cell (col, row).
func FlatHexCellCenter(col, row int) (x, y int) {
	y = row*FlatHexSpacingY + 3
	if col%2 == 1 {
		y += 3
	}
	return 7 + col*FlatHexSpacingX, y
}

// NearestCell finds the cell whose center is closest to world position (wx, wy).
func NearestCell(wx, wy, cols, rows int, centerFn CenterFunc) (col, row int) {
	bestDist := math.MaxFloat64
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cx, cy := centerFn(c, r)
			dx := float64(wx - cx)
			dy := float64(wy - cy)
			d := dx*dx + dy*dy
			if d < bestDist {
				bestDist = d
				col = c
				row = r
			}
		}
	}
	return col, row
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
