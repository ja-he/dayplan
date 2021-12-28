package colors

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/lucasb-eyer/go-colorful"
)

func ColorFromHexString(s string) tcell.Color {
	switch len(s) {
	case 7:
		s = s[1:] // trim preceding '#'
	case 6:

	default:
		panic("not 6 or 7 chars for a hex color string?!")
	}

	// TODO: errors
	r, _ := strconv.ParseInt(s[0:2], 16, 32)
	g, _ := strconv.ParseInt(s[2:4], 16, 32)
	b, _ := strconv.ParseInt(s[4:6], 16, 32)

	full := int32((r << 16) | (g << 8) | (b))

	return tcell.NewHexColor(full)
}

// TODO: bring in line with how Lighten works, i.E. make parameter (currently
//       0..100 but subj to change to 0..1 imo) actually define what "distance"
//       to black to cover, so on 100% darken it turns black, on 50% darken it
//       goes 50% darker, etc.
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

// Lightens a background color by the percentage provided.
// If a colors lightness (per HSL) is 60%, there are 40% of lightness
// to "cover" until fully lit. Lightening such a color by 25% would cover
// 25% of that 40%, i.E: 10%, making the resulting colors lightness 70%.
func Lighten(color tcell.Color, percentage int) tcell.Color {
	hex := color.Hex()
	r := (hex & 0x00ff0000) >> 16
	g := (hex & 0x0000ff00) >> 8
	b := (hex & 0x000000ff)

	hue, sat, ltn := colorful.Color{R: float64(r) / 255.0, G: float64(g) / 255.0, B: float64(b) / 255.0}.Hsl()
	if ltn > 1.0 {
		panic("lightness is huge?!")
	}

	scalar := float64(percentage) / 100.0

	lightnessDelta := 1.0 - ltn
	newLightness := ltn + (lightnessDelta * scalar)

	newColor := colorful.Hsl(hue, sat, newLightness)
	newR, newG, newB := newColor.RGB255()

	return tcell.NewHexColor(int32((int32(newR) << 16) | (int32(newG) << 8) | (int32(newB))))
}

func DarkenBG(style tcell.Style, percentage int) tcell.Style {
	_, bg, _ := style.Decompose()
	return style.Background(Darken(bg, percentage))
}

// Lightens the background of a given color by the provvided percentage.
func LightenBG(style tcell.Style, percentage int) tcell.Style {
	_, bg, _ := style.Decompose()
	return style.Background(Lighten(bg, percentage))
}

// Lightens the foreground of a given color by the provvided percentage.
func LightenFG(style tcell.Style, percentage int) tcell.Style {
	fg, _, _ := style.Decompose()
	return style.Foreground(Lighten(fg, percentage))
}

// Lightens both foreground and background of the provided color by 50%, which
// has a dimming or deemphasizing visual effect.
// TODO: if we want to parameterize this by terminal background color or how
//       much dim a user might have set in some option it would probably belong
//       in the model as a convenience function for the view (mostly)?
func DefaultDim(style tcell.Style) tcell.Style {
	fg, bg, _ := style.Decompose()
	return style.Foreground(Lighten(fg, 50)).Background(Lighten(bg, 50))
}

// Darkens the background of the provided color by 90%, which has an
// emphasizing visual effect.
// TODO: if we want to parameterize this by terminal background color or how
//       much dim a user might have set in some option it would probably belong
//       in the model as a convenience function for the view (mostly)?
func DefaultEmphasize(style tcell.Style) tcell.Style {
	return DarkenBG(style, 90) // TODO: dial in value perhaps
}
