package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/styling"
)

type TUIStyling struct {
	style tcell.Style
}

func (s *TUIStyling) AsTcell() tcell.Style { return s.style }
func (s *TUIStyling) DefaultDimmed() styling.DrawStyling {
	newStyle := styling.DefaultDim(s.style)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) DefaultEmphasized() styling.DrawStyling {
	newStyle := styling.DefaultEmphasize(s.style)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) Italicized() styling.DrawStyling {
	newStyle := s.style.Italic(true)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) Bolded() styling.DrawStyling {
	newStyle := s.style.Bold(true)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) LightenedFG(percentage int) styling.DrawStyling {
	newStyle := styling.LightenFG(s.style, percentage)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) LightenedBG(percentage int) styling.DrawStyling {
	newStyle := styling.LightenBG(s.style, percentage)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) DarkenedFG(percentage int) styling.DrawStyling {
	newStyle := styling.DarkenFG(s.style, percentage)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) DarkenedBG(percentage int) styling.DrawStyling {
	newStyle := styling.DarkenBG(s.style, percentage)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) ToString() string {
	tcellColorToString := func(color tcell.Color) string {
		return fmt.Sprintf("#%06x", color.Hex())
	}
	fg, bg, _ := s.style.Decompose()
	return fmt.Sprintf("(%s,%s)", tcellColorToString(fg), tcellColorToString(bg))
}

func FromTcell(style tcell.Style) styling.DrawStyling { return &TUIStyling{style: style} }
func NewStyling(fg, bg tcell.Color) styling.DrawStyling {
	return &TUIStyling{style: tcell.StyleDefault.Foreground(fg).Background(bg)}
}
