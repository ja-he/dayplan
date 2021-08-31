package colors

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
)

func ColorFromHexString(s string) tcell.Color {
	if len(s) != 6 {
		panic(fmt.Sprintf("string wrong size '%s'", s))
	}

	// TODO: errors
	r, _ := strconv.ParseInt(s[0:2], 16, 32)
	g, _ := strconv.ParseInt(s[2:4], 16, 32)
	b, _ := strconv.ParseInt(s[4:6], 16, 32)

	full := int32((r << 16) | (g << 8) | (b))

	return tcell.NewHexColor(full)
}

func Darken(color tcell.Color, percentage int) tcell.Color {
	hex := color.Hex()
	r := (hex & 0x00ff0000) >> 16
	g := (hex & 0x0000ff00) >> 8
	b := (hex & 0x000000ff)

	scalar := float64(percentage) / 100.0
	newR := int32(float64(r) * scalar)
	newG := int32(float64(g) * scalar)
	newB := int32(float64(b) * scalar)

	newHex := (newR << 16) | (newG << 8) | (newB)

	return tcell.NewHexColor(newHex)
}

func DarkenBG(style tcell.Style, percentage int) tcell.Style {
	_, bg, _ := style.Decompose()
	return style.Background(Darken(bg, percentage))
}
