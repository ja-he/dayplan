package tui

import "github.com/gdamore/tcell/v2"

import (
	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/colors"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

var errorCategoryStyle = tcell.StyleDefault.Background(tcell.ColorIndianRed)

type ToolsPanel struct {
	renderer *TUIRenderer

	currentCategory *model.Category
	categories      *category_style.CategoryStyling

	horizPadding, vertPadding, gap int

	lastBoxesDrawn map[model.Category]util.Rect
}

func (p *ToolsPanel) Draw(x, y, w, h int) {
	boxes := p.getCategoryBoxes(x, y, w, h)
	for cat, box := range boxes {
		style, err := p.categories.GetStyle(cat)
		if err != nil {
			style = errorCategoryStyle
		}

		textHeightOffset := box.H / 2
		textLen := box.W - 2
		p.renderer.DrawBox(style, box.X, box.Y, box.W, box.H)
		p.renderer.DrawText(box.X+1, box.Y+textHeightOffset, textLen, 0, style, util.TruncateAt(cat.Name, textLen))
		if p.currentCategory.Name == cat.Name {
			p.renderer.DrawBox(colors.DefaultEmphasize(style), box.X+box.W-1, box.Y, 1, box.H)
		}
	}
	p.lastBoxesDrawn = boxes
}

func (p *ToolsPanel) getCategoryBoxes(x, y, w, h int) map[model.Category]util.Rect {
	i := 0

	result := make(map[model.Category]util.Rect)

	for _, styling := range p.categories.GetAll() {
		box := util.Rect{
			X: x + p.horizPadding,
			Y: p.vertPadding + i + (i * p.gap),
			W: w - (2 * p.horizPadding),
			H: 1,
		}
		result[styling.Cat] = box
		i++
	}
	return result
}

func (p *ToolsPanel) getCategoryForpos(x, y int) *model.Category {
	for cat, box := range p.lastBoxesDrawn {
		if box.Contains(x, y) {
			return &cat
		}
	}
	return nil
}

func (p *ToolsPanel) GetPositionInfo(x, y int) ui.PositionInfo {
	return &TUIPositionInfo{
		paneType: ui.ToolsUIPanelType,
		weather:  ui.WeatherPanelPositionInfo{},
		timeline: ui.TimelinePanelPositionInfo{},
		tools:    ui.ToolsPanelPositionInfo{Category: p.getCategoryForpos(x, y)},
		status:   ui.StatusPanelPositionInfo{},
		events:   ui.EventsPanelPositionInfo{},
	}
}
