package styling

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/lucasb-eyer/go-colorful"
)

func colorfulColorToTcellColor(color colorful.Color) tcell.Color {
	r, g, b := color.RGB255()

	rgb := ((uint32(r)) << 16) | (uint32(g) << 8) | (uint32(b))

	return tcell.NewHexColor(int32(rgb))
}

func lightenColorfulColor(color colorful.Color, percentage int) colorful.Color {
	hue, sat, ltn := color.Hsl()
	if ltn > 1.0 {
		panic("lightness is huge?!")
	}

	scalar := float64(percentage) / 100.0

	lightnessDelta := 1.0 - ltn
	newLightness := ltn + (lightnessDelta * scalar)

	return colorful.Hsl(hue, sat, newLightness)
}

func darkenColorfulColor(color colorful.Color, percentage int) colorful.Color {
	hue, sat, ltn := color.Hsl()
	if ltn > 1.0 {
		panic("lightness is huge?!")
	}

	scalar := float64(percentage) / 100.0

	darknessDelta := ltn
	newLightness := ltn - (darknessDelta * scalar)

	return colorful.Hsl(hue, sat, newLightness)
}

// smartOffsetLuminanceBy returns for the given color a color with luminance
// changed from the original by the given luminance delta, either lightening or
// darkening the color, depending on the initial luminance value being below or
// above a luminance threshold.
func smartOffsetLuminanceBy(color colorful.Color, luminanceDelta float64) colorful.Color {
	hue, sat, lum := color.Clamped().HSLuv()

	if lum < 0.5 {
		lum += luminanceDelta
	} else {
		lum -= luminanceDelta
	}

	return colorful.HSLuv(hue, sat, lum)
}

func getLuminance(color colorful.Color) float64 {
	_, _, l := color.HSLuv()
	return l
}

func colorfulColorFromHexString(hex string) colorful.Color {
	color, err := colorful.Hex(hex)
	if err != nil {
		panic(fmt.Sprintf("unable to create colorful.Color from '%s' due to error: '%s'", hex, err.Error()))
	}
	return color
}
