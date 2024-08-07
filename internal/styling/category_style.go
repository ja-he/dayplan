package styling

// TODO(ja-he):
//   This file contains multiple orthogonal functionalities and should probably
//   be split up or removed entirely, e.g., such that:
//     - the model has a CategoryHandler interface
//     - it also implements CategoryStyling, which has a Get(cat)styling method

import (
	"fmt"

	"github.com/ja-he/dayplan/internal/model"
)

// StyledCategory is a category associated with a styling by which it should be
// rendered.
type StyledCategory struct {
	Style DrawStyling
	Cat   model.Category
}

// CategoryStyling is a set of styled categories.
type CategoryStyling struct {
	styles []StyledCategory
}

// GetKnownCategoriesByName returns a mapping of category names to the fully
// parameterized categories (i.E. including priority), provided they exist.
// Allows ensuring that category data stays consistent across the program.
func (cs *CategoryStyling) GetKnownCategoriesByName() map[model.CategoryName]*model.Category {
	result := make(map[model.CategoryName]*model.Category)

	for i := range cs.styles {
		cat := &cs.styles[i].Cat
		result[cat.Name] = cat
	}

	return result
}

// Add adds the given styling for the given category to this CategoryStyling.
func (cs *CategoryStyling) Add(cat model.Category, style DrawStyling) {
	styling := StyledCategory{Cat: cat, Style: style}
	cs.styles = append(cs.styles, styling)
}

// EmptyCategoryStyling returns an empty category styling.
func EmptyCategoryStyling() *CategoryStyling {
	cs := CategoryStyling{}
	return &cs
}

// GetAll returns all styled categories in this styling.
func (cs *CategoryStyling) GetAll() []StyledCategory {
	return cs.styles
}

// GetStyle returns the styling for the requested category from this styling.
//
// If no styling is present for the category, it returns nil an an error.
func (cs *CategoryStyling) GetStyle(c model.CategoryName) (DrawStyling, error) {
	for _, styling := range cs.styles {
		if styling.Cat.Name == c {
			return styling.Style, nil
		}
	}
	return nil, fmt.Errorf("style for category '%s' not found", c)
}
