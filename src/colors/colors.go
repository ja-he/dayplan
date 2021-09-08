package colors

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/lucasb-eyer/go-colorful"
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

	hue, sat, ltn := colorful.Color{R: float64(r) / 255.0, G: float64(g) / 255.0, B: float64(b) / 255.0}.Hsl()

	scalar := float64(percentage) / 100.0

	newColor := colorful.Hsl(hue, sat, ltn*scalar)
	newR, newG, newB := newColor.RGB255()

	return tcell.NewHexColor(int32((int32(newR) << 16) | (int32(newG) << 8) | (int32(newB))))

}

func DarkenBG(style tcell.Style, percentage int) tcell.Style {
	_, bg, _ := style.Decompose()
	return style.Background(Darken(bg, percentage))
}
