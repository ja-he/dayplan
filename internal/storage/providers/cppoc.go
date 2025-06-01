package providers

import "github.com/ja-he/dayplan/internal/model"

// TODO: move somewhere / rename
type CPPOC struct {
	M map[model.CategoryName]*model.Category
}

func (c *CPPOC) GetCategory(name model.CategoryName) *model.Category {
	cat, ok := c.M[name]
	if !ok {
		return nil
	}
	return cat
}
