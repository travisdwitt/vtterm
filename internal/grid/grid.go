package grid

import "strings"

const (
	cellWidth  = 7 // inner width so tokens (5 wide) sit centered
	cellHeight = 3 // inner height so tokens (3 tall) sit centered
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
