package tui

import (
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

type ToolsPane struct {
	renderer ui.ConstrainedRenderer

	dimensions func() (x, y, w, h int)

	stylesheet styling.Stylesheet

	currentCategory *model.Category
	categories      *styling.CategoryStyling

	horizPadding, vertPadding, gap int

	lastBoxesDrawn map[model.Category]util.Rect
}

func (p *ToolsPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *ToolsPane) Draw() {
	x, y, w, h := p.dimensions()

	boxes := p.getCategoryBoxes(x, y, w, h)
	for cat, box := range boxes {
		categoryStyle, err := p.categories.GetStyle(cat)
		var styling styling.DrawStyling
		if err != nil {
			styling = p.stylesheet.CategoryFallback()
		} else {
			styling = categoryStyle
		}

		textHeightOffset := box.H / 2
		textLen := box.W - 2
		p.renderer.DrawBox(box.X, box.Y, box.W, box.H, styling)
		p.renderer.DrawText(box.X+1, box.Y+textHeightOffset, textLen, 1, styling, util.TruncateAt(cat.Name, textLen))
		if p.currentCategory.Name == cat.Name {
			p.renderer.DrawBox(box.X+box.W-1, box.Y, 1, box.H, styling.DefaultEmphasized())
		}
	}
	p.lastBoxesDrawn = boxes
}

func (p *ToolsPane) getCategoryBoxes(x, y, w, h int) map[model.Category]util.Rect {
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

func (p *ToolsPane) getCategoryForpos(x, y int) *model.Category {
	for cat, box := range p.lastBoxesDrawn {
		if box.Contains(x, y) {
			return &cat
		}
	}
	return nil
}

func (p *ToolsPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return &TUIPositionInfo{
		paneType: ui.ToolsUIPaneType,
		weather:  nil,
		timeline: nil,
		tools:    &ToolsPanePositionInfo{category: p.getCategoryForpos(x, y)},
		status:   nil,
		events:   nil,
	}
}

type ToolsPanePositionInfo struct {
	category *model.Category
}

func (i *ToolsPanePositionInfo) Category() *model.Category { return i.category }
