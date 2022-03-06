package styling

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

type TUIStyling struct {
	style tcell.Style
}

func (s *TUIStyling) AsTcell() tcell.Style { return s.style }
func (s *TUIStyling) DefaultDimmed() DrawStyling {
	newStyle := DefaultDim(s.style)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) DefaultEmphasized() DrawStyling {
	newStyle := DefaultEmphasize(s.style)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) Italicized() DrawStyling {
	newStyle := s.style.Italic(true)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) Bolded() DrawStyling {
	newStyle := s.style.Bold(true)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) LightenedFG(percentage int) DrawStyling {
	newStyle := LightenFG(s.style, percentage)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) LightenedBG(percentage int) DrawStyling {
	newStyle := LightenBG(s.style, percentage)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) DarkenedFG(percentage int) DrawStyling {
	newStyle := DarkenFG(s.style, percentage)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) DarkenedBG(percentage int) DrawStyling {
	newStyle := DarkenBG(s.style, percentage)
	return &TUIStyling{style: newStyle}
}
func (s *TUIStyling) ToString() string {
	tcellColorToString := func(color tcell.Color) string {
		return fmt.Sprintf("#%06x", color.Hex())
	}
	fg, bg, _ := s.style.Decompose()
	return fmt.Sprintf("(%s,%s)", tcellColorToString(fg), tcellColorToString(bg))
}

func FromTcell(style tcell.Style) DrawStyling { return &TUIStyling{style: style} }
func NewTUIStyling(fg, bg tcell.Color) DrawStyling {
	return &TUIStyling{style: tcell.StyleDefault.Foreground(fg).Background(bg)}
}
