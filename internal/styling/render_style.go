package styling

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/internal/config"
	"github.com/lucasb-eyer/go-colorful"
)

// DrawStyling is style information used for rendering text.
// It should represent foreground and background color as well as modifiers
// such as italicization.
// A DrawStyling can be converted to any styling needed by a renderer, e.g., AsTcell
// tcell.Style for a tcell-based renderer via AsTcell.
type DrawStyling interface {
	AsTcell() tcell.Style

	DefaultDimmed() DrawStyling
	DefaultEmphasized() DrawStyling
	NormalizeFromBG(luminanceOffset float64) DrawStyling
	Invert() DrawStyling
	LightenedFG(percentage int) DrawStyling
	LightenedBG(percentage int) DrawStyling
	DarkenedFG(percentage int) DrawStyling
	DarkenedBG(percentage int) DrawStyling

	Italicized() DrawStyling
	Unitalicized() DrawStyling
	Bolded() DrawStyling
	Unbolded() DrawStyling

	ToString() string
}

// FallbackStyling is a DrawStyling that holds non-renderer-specific colors.
type FallbackStyling struct {
	fg colorful.Color
	bg colorful.Color

	bold, italic, underlined bool
}

// AsTcell returns this styling as a tcell.Style.
func (s *FallbackStyling) AsTcell() tcell.Style {
	fg := colorfulColorToTcellColor(s.fg)
	bg := colorfulColorToTcellColor(s.bg)

	// convert colors
	style := tcell.StyleDefault.Foreground(fg).Background(bg)

	// set attributes
	style = style.Bold(s.bold).Italic(s.italic).Underline(s.underlined)

	return style
}

// DefaultDimmed returns a copy of this styling with 'dimmed' colors, i.E. it
// lightens them by a default value.
func (s *FallbackStyling) DefaultDimmed() DrawStyling {
	result := s.clone()
	result.fg = lightenColorfulColor(result.fg, 50)
	result.bg = lightenColorfulColor(result.bg, 50)
	return result
}

// DefaultEmphasized returns a copy of this styling with 'emphasized' colors,
// i.E. it darkens them by a default value.
func (s *FallbackStyling) DefaultEmphasized() DrawStyling {
	result := s.clone()
	result.fg = darkenColorfulColor(result.fg, 20)
	result.bg = darkenColorfulColor(result.bg, 20)
	return result
}

// Invert returns an inversion of the style.
func (s *FallbackStyling) Invert() DrawStyling {
	result := &FallbackStyling{}
	result.bg = s.fg
	result.fg = s.bg
	return result
}

// NormalizeFromBG returns a new style based off the background color where the
// foreground is set to an appropriate pairing color for the background.
func (s *FallbackStyling) NormalizeFromBG(luminanceOffset float64) DrawStyling {
	result := s.clone()
	result.fg = smartOffsetLuminanceBy(s.bg, luminanceOffset)
	return result
}

// LightenedFG returns a copy of this styling with the foreground color
// lightened by the requested percentage.
func (s *FallbackStyling) LightenedFG(percentage int) DrawStyling {
	result := s.clone()
	result.fg = lightenColorfulColor(result.fg, percentage)
	return result
}

// LightenedBG returns a copy of this styling with the background color
// lightened by the requested percentage.
func (s *FallbackStyling) LightenedBG(percentage int) DrawStyling {
	result := s.clone()
	result.bg = lightenColorfulColor(result.bg, percentage)
	return result
}

// DarkenedFG returns a copy of this styling with the foreground color darkened
// by the requested percentage.
func (s *FallbackStyling) DarkenedFG(percentage int) DrawStyling {
	result := s.clone()
	result.fg = darkenColorfulColor(result.fg, percentage)
	return result
}

// DarkenedBG returns a copy of this styling with the background color darkened
// by the requested percentage.
func (s *FallbackStyling) DarkenedBG(percentage int) DrawStyling {
	result := s.clone()
	result.bg = darkenColorfulColor(result.bg, percentage)
	return result
}

// Italicized returns a copy of this styling which is guaranteed to be
// italicized. If the original styling was already italicized, this effectively
// returns an exact copy.
func (s *FallbackStyling) Italicized() DrawStyling {
	result := s.clone()
	result.italic = true
	return result
}

// Unitalicized returns a copy of this styling which is guaranteed
// to _not_ be italicized.
// If the original styling was already not italicized, this
// effectively returns an exact copy.
func (s *FallbackStyling) Unitalicized() DrawStyling {
	result := s.clone()
	result.italic = false
	return result
}

// Unbolded returns a copy of this styling which is guaranteed to
// _not_ be bolded.
// If the original styling was already non-bolded, this effectively
// returns an exact copy.
func (s *FallbackStyling) Unbolded() DrawStyling {
	result := s.clone()
	result.bold = false
	return result
}

// Bolded returns a copy of this styling which is guaranteed to be
// bolded. If the original styling was already bolded, this effectively
// returns an exact copy.
func (s *FallbackStyling) Bolded() DrawStyling {
	result := s.clone()
	result.bold = true
	return result
}

// ToString returns a string representation of this styling, e.g., for logging
// purposes.
func (s *FallbackStyling) ToString() string {
	return fmt.Sprintf(
		"[fg:'%s' bg:'%s' (b:%t i:%t u:%t)]",
		s.fg.Hex(),
		s.bg.Hex(),
		s.bold,
		s.italic,
		s.underlined,
	)
}

func (s *FallbackStyling) clone() *FallbackStyling {
	newS := *s
	return &newS
}

// StyleFromHexPair constructs and returns a styling from two hexadecimally
// formatted strings for the foreground and background color.
// Strings have to have hexadecimal or HTML color notation and lead with a '#'.
//
// Examples:
//   - '#ff0000'
//   - '#fff'
//   - '#BEEF42'
func StyleFromHexPair(fg, bg string) *FallbackStyling {
	return &FallbackStyling{
		fg: colorfulColorFromHexString(fg),
		bg: colorfulColorFromHexString(bg),
	}
}

// StyleFromHexSingle takes the given hex color as a background and returns a
// style in which the foreground is inferred from the background (same hue and
// saturation, different luminance).
func StyleFromHexSingle(hexString string, darkBG bool) *FallbackStyling {
	accentColor := colorfulColorFromHexString(hexString)
	var bg, fg colorful.Color
	lum := getLuminance(accentColor)
	if darkBG && lum <= 0.5 || !darkBG && lum > 0.5 {
		bg = accentColor
		fg = smartOffsetLuminanceBy(accentColor, 0.5)
	} else {
		fg = accentColor
		bg = smartOffsetLuminanceBy(accentColor, 0.5)
	}
	return &FallbackStyling{
		fg: fg,
		bg: bg,
	}
}

// StyleFromColors constructs a style by the given colors.
func StyleFromColors(fg, bg colorful.Color) *FallbackStyling {
	return &FallbackStyling{
		fg: fg,
		bg: bg,
	}
}

// StyleFromConfig takes a styling as specified in a configuration file and
// converts it to a usable DrawStyling.
func StyleFromConfig(config config.Styling) DrawStyling {
	styling := StyleFromHexPair(config.Fg, config.Bg)
	if config.Style != nil {
		styling.bold = config.Style.Bold
		styling.italic = config.Style.Italic
		styling.underlined = config.Style.Underlined
	}
	return styling
}
