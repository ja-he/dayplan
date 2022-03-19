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

func colorfulColorFromHexString(hex string) colorful.Color {
	color, err := colorful.Hex(hex)
	if err != nil {
		panic(fmt.Sprintf("unable to create colorful.Color from '%s' due to error: '%s'", hex, err.Error()))
	}
	return color
}
