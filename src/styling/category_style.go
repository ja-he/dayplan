package styling

// TODO(ja-he):
//   This file contains multiple orthogonal functionalities and should probably
//   be split up or removed entirely, e.g., such that:
//     - the model has a CategoryHandler interface
//     - it also implements CategoryStyling, which has a Get(cat)styling method

import (
	"fmt"
	"io/ioutil"

	"github.com/ja-he/dayplan/src/model"

	"gopkg.in/yaml.v2"
)

type styledCategory struct {
	Style DrawStyling
	Cat   model.Category
}

type CategoryStyling struct {
	styles []styledCategory
}

// represented as YAML in category style file
type styledCategoryInput struct {
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

func (cs *CategoryStyling) Add(cat model.Category, style DrawStyling) {
	styling := styledCategory{Cat: cat, Style: style}
	cs.styles = append(cs.styles, styling)
}

// TODO: move
func ReadCategoryStylingFile(filepath string) ([]styledCategoryInput, error) {
	result := []styledCategoryInput{}

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

func StyleFromHex(fg, bg string) DrawStyling {
	return NewTUIStyling(ColorFromHexString(fg), ColorFromHexString(bg))
}

func (cs *CategoryStyling) AddStyleFromInput(input styledCategoryInput) bool {
	cat := model.Category{
		Name:     input.Name,
		Priority: input.Priority,
	}
	style := StyleFromHex(input.Fg, input.Bg)

	// TODO: error checking e.g. for the colors (which currently would panic)

	cs.Add(cat, style)
	return true
}

func (cs *CategoryStyling) GetAll() []styledCategory {
	return cs.styles
}

func (cs *CategoryStyling) GetStyle(c model.Category) (DrawStyling, error) {
	for _, styling := range cs.styles {
		if styling.Cat.Name == c.Name {
			return styling.Style, nil
		}
	}
	return nil, fmt.Errorf("style for category '%s' not found", c.Name)
}
