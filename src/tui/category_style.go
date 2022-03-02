package tui

// TODO(ja-he):
//   This file contains multiple orthogonal functionalities and should be split
//   up.
//   It also does not belong in the TUI package and is here temporarily until
//   customizable styling is properly implemented (in whatever shape that will
//   take).

import (
	"fmt"
	"io/ioutil"

	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"

	// TODO: this shouldnt use tcell, but use the styling interface instead
	"github.com/gdamore/tcell/v2"
	"gopkg.in/yaml.v2"
)

type StyledCategory struct {
	Style tcell.Style
	Cat   model.Category
}

type CategoryStyling struct {
	styles []StyledCategory
}

// represented as YAML in category style file
type StyledCategoryInput struct {
	Name     string
	Fg       string
	Bg       string
	Priority int
}

// Returns a mapping of category names to the fully parameterized categories
// (i.E. including priority), provided they exist.
// Allows ensuring that category data stays consistent across the program.
func (cs *CategoryStyling) GetKnownCategoriesByName() map[string]*model.Category {
	result := make(map[string]*model.Category)

	for i := range cs.styles {
		cat := &cs.styles[i].Cat
		result[cat.Name] = cat
	}

	return result
}

func (cs *CategoryStyling) Add(cat model.Category, style tcell.Style) {
	styling := StyledCategory{Cat: cat, Style: style}
	cs.styles = append(cs.styles, styling)
}

func ReadCategoryStylingFile(filepath string) ([]StyledCategoryInput, error) {
	result := []StyledCategoryInput{}

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal([]byte(data), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func EmptyCategoryStyling() *CategoryStyling {
	cs := CategoryStyling{}
	return &cs
}

func DefaultCategoryStyling() *CategoryStyling {
	cs := CategoryStyling{}

	cs.Add(model.Category{Name: "misc"}, tcell.StyleDefault.Background(tcell.Color250).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "work"}, tcell.StyleDefault.Background(tcell.NewHexColor(0xccebff)).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "leisure"}, tcell.StyleDefault.Background(tcell.Color76).Foreground(tcell.ColorReset))
	cs.Add(model.Category{Name: "fitness"}, tcell.StyleDefault.Background(tcell.Color208).Foreground(tcell.ColorReset))

	return &cs
}

func StyleFromHex(fg, bg string) tcell.Style {
	style := tcell.StyleDefault
	style = style.Foreground(styling.ColorFromHexString(fg))
	style = style.Background(styling.ColorFromHexString(bg))
	return style
}

func (cs *CategoryStyling) AddStyleFromInput(input StyledCategoryInput) bool {
	cat := model.Category{
		Name:     input.Name,
		Priority: input.Priority,
	}
	style := StyleFromHex(input.Fg, input.Bg)

	// TODO: error checking e.g. for the colors (which currently would panic)

	cs.Add(cat, style)
	return true
}

func (cs *CategoryStyling) GetAll() []StyledCategory {
	return cs.styles
}

func (cs *CategoryStyling) GetStyle(c model.Category) (tcell.Style, error) {
	for _, styling := range cs.styles {
		if styling.Cat.Name == c.Name {
			return styling.Style, nil
		}
	}
	return tcell.StyleDefault, fmt.Errorf("style for category '%s' not found", c.Name)
}
