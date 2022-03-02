package styling

import "github.com/gdamore/tcell/v2"

// DrawStyling is style information used for rendering text.
// It should represent foreground and background color as well as modifiers
// such as italicization.
// A DrawStyling can be converted to any styling needed by a renderer, e.g.,
// tcell.Style for a tcell-based renderer via AsTcell.
type DrawStyling interface {
	AsTcell() tcell.Style

	DefaultDimmed() DrawStyling
	DefaultEmphasized() DrawStyling
	LightenedFG(percentage int) DrawStyling
	LightenedBG(percentage int) DrawStyling
	DarkenedFG(percentage int) DrawStyling
	DarkenedBG(percentage int) DrawStyling

	Italicized() DrawStyling
	Bolded() DrawStyling
}
