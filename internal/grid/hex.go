package grid

import "strings"

func RenderPointyHex(cols, rows int) string {
	hexSpacing := 12
	halfSpacing := hexSpacing / 2

	bufW := cols*hexSpacing + halfSpacing + 3
	bufH := rows*4 + 1

	buf := newBuffer(bufW, bufH)

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			x := halfSpacing + col*hexSpacing
			y := row * 4
			stampPointy(buf, x, y)
		}
	}

	return buf.String()
}

func stampPointy(buf *charBuffer, x, y int) {
	buf.set(x, y, '>')
	buf.set(x+1, y, '-')
	buf.set(x+2, y, '<')
	buf.set(x-2, y+1, '/')
	buf.set(x+4, y+1, '\\')
	buf.set(x-6, y+2, '>')
	buf.set(x-5, y+2, '-')
	buf.set(x-4, y+2, '<')
	buf.set(x+6, y+2, '>')
	buf.set(x+7, y+2, '-')
	buf.set(x+8, y+2, '<')
	buf.set(x-2, y+3, '\\')
	buf.set(x+4, y+3, '/')
	buf.set(x, y+4, '>')
	buf.set(x+1, y+4, '-')
	buf.set(x+2, y+4, '<')
}

func RenderFlatHex(cols, rows int) string {
	bufW := cols*10 + 4
	bufH := rows*6 + 4

	buf := newBuffer(bufW, bufH)

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			tx := 3 + col*10
			ty := row * 6
			if col%2 == 1 {
				ty += 3
			}
			stampFlat(buf, tx, ty)
		}
	}

	return buf.String()
}

func stampFlat(buf *charBuffer, tx, ty int) {
	buf.set(tx, ty, '+')
	for i := 1; i <= 6; i++ {
		buf.set(tx+i, ty, '-')
	}
	buf.set(tx+7, ty, '+')

	buf.set(tx-1, ty+1, '/')
	buf.set(tx+8, ty+1, '\\')
	buf.set(tx-2, ty+2, '/')
	buf.set(tx+9, ty+2, '\\')

	buf.set(tx-3, ty+3, '+')
	buf.set(tx+10, ty+3, '+')

	buf.set(tx-2, ty+4, '\\')
	buf.set(tx+9, ty+4, '/')
	buf.set(tx-1, ty+5, '\\')
	buf.set(tx+8, ty+5, '/')

	buf.set(tx, ty+6, '+')
	for i := 1; i <= 6; i++ {
		buf.set(tx+i, ty+6, '-')
	}
	buf.set(tx+7, ty+6, '+')
}

type charBuffer struct {
	data [][]byte
	w, h int
}

func newBuffer(w, h int) *charBuffer {
	data := make([][]byte, h)
	for i := range data {
		data[i] = make([]byte, w)
		for j := range data[i] {
			data[i][j] = ' '
		}
	}
	return &charBuffer{data: data, w: w, h: h}
}

func (b *charBuffer) set(x, y int, c byte) {
	if x >= 0 && x < b.w && y >= 0 && y < b.h {
		b.data[y][x] = c
	}
}

func (b *charBuffer) String() string {
	var sb strings.Builder
	for _, row := range b.data {
		sb.WriteString(strings.TrimRight(string(row), " "))
		sb.WriteByte('\n')
	}
	return sb.String()
}
