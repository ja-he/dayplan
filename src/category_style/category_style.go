package category_style

import (
	"fmt"
	"strings"

	"dayplan/src/colors"
	"dayplan/src/model"

	"github.com/gdamore/tcell/v2"
)

type CategoryStyling struct {
	styles []StyledCat
}

func (cs *CategoryStyling) Add(cat model.Category, style tcell.Style) {
	styling := StyledCat{Cat: cat, Style: style}
	cs.styles = append(cs.styles, styling)
}

type StyledCat struct {
	Style tcell.Style
	Cat   model.Category
}

func EmptyCategoryStyling() *CategoryStyling {
	cs := CategoryStyling{}
	return &cs
}

func DefaultCategoryStyling() *CategoryStyling {
	cs := CategoryStyling{}

	cs.Add(model.Category{Name: "default"}, tcell.StyleDefault.Background(tcell.NewHexColor(0xff00ff)).Foreground(tcell.NewHexColor(0x00ff00)))
	cs.Add(model.Category{Name: "work"}, tcell.StyleDefault.Background(tcell.NewHexColor(0xccebff)).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "leisure"}, tcell.StyleDefault.Background(tcell.Color76).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "misc"}, tcell.StyleDefault.Background(tcell.Color250).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "programming"}, tcell.StyleDefault.Background(tcell.Color226).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "cooking"}, tcell.StyleDefault.Background(tcell.Color212).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "fitness"}, tcell.StyleDefault.Background(tcell.Color208).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "eating"}, tcell.StyleDefault.Background(tcell.Color224).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "hygiene"}, tcell.StyleDefault.Background(tcell.Color80).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "cleaning"}, tcell.StyleDefault.Background(tcell.Color215).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "laundry"}, tcell.StyleDefault.Background(tcell.Color111).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "family"}, tcell.StyleDefault.Background(tcell.Color122).Foreground(tcell.ColorReset))

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

	cs.Add(cat, style)

	return true
}

func (cs *CategoryStyling) GetAll() []StyledCat {
	return cs.styles
}

func (cs *CategoryStyling) GetStyle(c model.Category) tcell.Style {
	for _, styling := range cs.styles {
		if styling.Cat == c {
			return styling.Style
		}
	}
	panic(fmt.Sprintf("unknown style for category '%s' requested", c.Name))
}
