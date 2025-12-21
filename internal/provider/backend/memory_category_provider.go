package backend

import (
	"fmt"
	"sync"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/styling"
)

// TODO: move somewhere / rename
type MemoryCategoryProvider struct {
	mtx sync.RWMutex

	M map[model.CategoryName]*model.Category
}

func (c *MemoryCategoryProvider) GetCategory(name model.CategoryName) *model.Category {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	cat, ok := c.M[name]
	if !ok {
		return nil
	}
	return cat
}

func GetCategoriesByNameFromConfig(configData config.Config) (map[model.CategoryName]*model.Category, error) {
	categoriesByName := map[model.CategoryName]*model.Category{}
	for _, category := range configData.Categories {

		var goal model.Goal
		var err error
		switch {
		case category.Goal.Ranged != nil:
			goal, err = model.NewRangedGoalFromConfig(*category.Goal.Ranged)
		case category.Goal.Workweek != nil:
			goal, err = model.NewWorkweekGoalFromConfig(*category.Goal.Workweek)
		}
		if err != nil {
			return nil, fmt.Errorf("Unable to parse goal: %w", err)
		}

		color, err := styling.ColorfulColorFromHexString(category.Color)
		if err != nil {
			return nil, fmt.Errorf("could not parse color from hex string: %w", err)
		}
		cat := model.Category{
			Name:       model.CategoryName(category.Name),
			Priority:   category.Priority,
			Goal:       goal,
			Deprecated: category.Deprecated,
			Color:      color,
		}
		categoriesByName[model.CategoryName(category.Name)] = &cat
	}
	return categoriesByName, nil
}
