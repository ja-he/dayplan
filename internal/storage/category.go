package storage

import "github.com/ja-he/dayplan/internal/model"

type CategoryProvider interface {
	GetCategory(model.CategoryName) *model.Category
}
