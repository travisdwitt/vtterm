package grid

import "strings"

const (
	cellWidth  = 3 // matches token label width (5 total with +…+ borders)
	cellHeight = 1 // matches token label height (3 total with +…+ borders)
)

func RenderSquare(cols, rows int) string {
	var b strings.Builder

	for row := 0; row <= rows; row++ {
		for col := 0; col <= cols; col++ {
			b.WriteByte('+')
			if col < cols {
				b.WriteString(strings.Repeat("-", cellWidth))
			}
		}
		b.WriteByte('\n')

		if row < rows {
			for line := 0; line < cellHeight; line++ {
				for col := 0; col <= cols; col++ {
					b.WriteByte('|')
					if col < cols {
						b.WriteString(strings.Repeat(" ", cellWidth))
					}
				}
				b.WriteByte('\n')
			}
		}
	}

	return b.String()
}
