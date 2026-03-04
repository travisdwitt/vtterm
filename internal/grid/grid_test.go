package grid

import (
	"fmt"
	"strings"
	"testing"
)

func TestRenderSquare(t *testing.T) {
	out := RenderSquare(3, 2)
	if !strings.Contains(out, "+--------+") {
		t.Error("expected grid border")
	}
}

func TestRenderFlatHex(t *testing.T) {
	out := RenderFlatHex(4, 3)
	fmt.Print(out)
	if !strings.Contains(out, "+------+") {
		t.Error("expected flat hex edge")
	}
}
