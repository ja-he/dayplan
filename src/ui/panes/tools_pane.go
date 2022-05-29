package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

// ToolsPane shows tools for editing.
// Currently it only offers a selection of categories to select from.
type ToolsPane struct {
	InputProcessingLeafPane

	currentCategory *model.Category
	categories      *styling.CategoryStyling

	horizPadding, vertPadding, gap int

	lastBoxesDrawn map[model.Category]util.Rect
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *ToolsPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw draws this pane.
func (p *ToolsPane) Draw() {
	x, y, w, h := p.dimensions()

	style := p.stylesheet.Normal
	if p.HasFocus() {
		style = p.stylesheet.NormalEmphasized
	}
	p.renderer.DrawBox(x, y, w, h, style)

	boxes := p.getCategoryBoxes(x, y, w, h)
	for cat, box := range boxes {
		categoryStyle, err := p.categories.GetStyle(cat)
		var styling styling.DrawStyling
		if err != nil {
			styling = p.stylesheet.CategoryFallback
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

func (p *ToolsPane) getCategoryForPos(x, y int) *model.Category {
	for cat, box := range p.lastBoxesDrawn {
		if box.Contains(x, y) {
			return &cat
		}
	}
	return nil
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *ToolsPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return ui.NewPositionInfo(
		ui.ToolsPaneType,
		nil,
		nil,
		&ToolsPanePositionInfo{category: p.getCategoryForPos(x, y)},
		nil,
		nil,
	)
}

// ToolsPanePositionInfo conveys information on a position in a tools pane,
// importantly the possible category displayed at that position.
type ToolsPanePositionInfo struct {
	category *model.Category
}

// Category gives the category at the position, or nil if none (e.g., because
// in padding space).
func (i *ToolsPanePositionInfo) Category() *model.Category { return i.category }

// NewToolsPane constructs and returns a new ToolsPane.
func NewToolsPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	inputProcessor input.ModalInputProcessor,
	currentCategory *model.Category,
	categories *styling.CategoryStyling,
	horizPadding int,
	vertPadding int,
	gap int,
) *ToolsPane {
	return &ToolsPane{
		InputProcessingLeafPane: InputProcessingLeafPane{
			InputProcessingPaneBaseData: InputProcessingPaneBaseData{
				ID:             ui.GeneratePaneID(),
				InputProcessor: inputProcessor,
			},
			renderer:   renderer,
			dimensions: dimensions,
			stylesheet: stylesheet,
		},
		currentCategory: currentCategory,
		categories:      categories,
		horizPadding:    horizPadding,
		vertPadding:     vertPadding,
		gap:             gap,
	}
}
