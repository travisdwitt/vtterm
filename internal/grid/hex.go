package grid

import "strings"

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
