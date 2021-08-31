package category_style

import (
	"strings"

	"dayplan/model"
	"dayplan/colors"

	"github.com/gdamore/tcell/v2"
)

type CategoryStyling struct {
	styles map[model.Category]tcell.Style
}

func EmptyCategoryStyling() *CategoryStyling {
	cs := CategoryStyling{}
	cs.styles = make(map[model.Category]tcell.Style)
  return &cs
}

func DefaultCategoryStyling() *CategoryStyling {
	cs := CategoryStyling{}
	cs.styles = make(map[model.Category]tcell.Style)

	cs.styles[model.Category{Name: "default"}] = tcell.StyleDefault.Background(tcell.NewHexColor(0xff00ff)).Foreground(tcell.NewHexColor(0x00ff00))
	cs.styles[model.Category{Name: "work"}] = tcell.StyleDefault.Background(tcell.NewHexColor(0xccebff)).Foreground(tcell.ColorReset)
	cs.styles[model.Category{Name: "leisure"}] = tcell.StyleDefault.Background(tcell.Color76).Foreground(tcell.ColorReset)
	cs.styles[model.Category{Name: "misc"}] = tcell.StyleDefault.Background(tcell.Color250).Foreground(tcell.ColorReset)
	cs.styles[model.Category{Name: "programming"}] = tcell.StyleDefault.Background(tcell.Color226).Foreground(tcell.ColorReset)
	cs.styles[model.Category{Name: "cooking"}] = tcell.StyleDefault.Background(tcell.Color212).Foreground(tcell.ColorReset)
	cs.styles[model.Category{Name: "fitness"}] = tcell.StyleDefault.Background(tcell.Color208).Foreground(tcell.ColorReset)
	cs.styles[model.Category{Name: "eating"}] = tcell.StyleDefault.Background(tcell.Color224).Foreground(tcell.ColorReset)
	cs.styles[model.Category{Name: "hygiene"}] = tcell.StyleDefault.Background(tcell.Color80).Foreground(tcell.ColorReset)
	cs.styles[model.Category{Name: "cleaning"}] = tcell.StyleDefault.Background(tcell.Color215).Foreground(tcell.ColorReset)
	cs.styles[model.Category{Name: "laundry"}] = tcell.StyleDefault.Background(tcell.Color111).Foreground(tcell.ColorReset)
	cs.styles[model.Category{Name: "family"}] = tcell.StyleDefault.Background(tcell.Color122).Foreground(tcell.ColorReset)

	return &cs
}

func StyleFromHex(fg, bg string) tcell.Style {
	style := tcell.StyleDefault
	style = style.Foreground(colors.ColorFromHexString(fg))
	style = style.Background(colors.ColorFromHexString(bg))
	return style
}

func (cs *CategoryStyling) AddStyleFromCfg(s string) bool {
	tokens := strings.SplitN(s, "|", 3)
	if len(tokens) != 3 {
		return false
	}

	name := tokens[0]
  fgstr := tokens[1][1:]
  bgstr := tokens[2][1:]

	cat := model.Category{Name: name}
	style := StyleFromHex(fgstr, bgstr)

	cs.styles[cat] = style

	return true
}

func (cs *CategoryStyling) GetStyle(c model.Category) tcell.Style {
	return cs.styles[c]
}
