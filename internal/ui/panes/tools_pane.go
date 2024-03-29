package panes

import (
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
)

// ToolsPane shows tools for editing.
// Currently it only offers a selection of categories to select from.
type ToolsPane struct {
	ui.LeafPane

	currentCategory *model.Category
	categories      *styling.CategoryStyling

	horizPadding, vertPadding, gap int

	lastBoxesDrawn map[model.Category]util.Rect
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *ToolsPane) Dimensions() (x, y, w, h int) {
	return p.Dims()
}

// Draw draws this pane.
func (p *ToolsPane) Draw() {
	if !p.IsVisible() {
		return
	}

	x, y, w, h := p.Dims()

	style := p.Stylesheet.Normal
	if p.HasFocus() {
		style = p.Stylesheet.NormalEmphasized
	}
	p.Renderer.DrawBox(x, y, w, h, style)

	// title
	func() {
		style := p.Stylesheet.NormalEmphasized.DefaultEmphasized()

		p.Renderer.DrawBox(x, y, w, 1, style)

		titleText := "Tools"
		p.Renderer.DrawText(x+(w/2)-(len(titleText)/2), y, len(titleText), 1, style.Bolded(), titleText)
	}()

	boxes := p.getCategoryBoxes(x, y+1, w, h)
	for cat, box := range boxes {
		categoryStyle, err := p.categories.GetStyle(cat)
		var styling styling.DrawStyling
		if err != nil {
			styling = p.Stylesheet.CategoryFallback
		} else {
			styling = categoryStyle
		}

		textHeightOffset := box.H / 2
		textLen := box.W - 2

		if p.currentCategory.Name == cat.Name {
			styling = styling.Invert().Bolded()
		}

		p.Renderer.DrawBox(box.X, box.Y, box.W, box.H, styling)
		p.Renderer.DrawText(box.X+1, box.Y+textHeightOffset, textLen, 1, styling, util.TruncateAt(cat.Name, textLen))
	}
	p.lastBoxesDrawn = boxes
}

func (p *ToolsPane) getCategoryBoxes(x, y, w, h int) map[model.Category]util.Rect {
	i := y

	result := make(map[model.Category]util.Rect)

	for _, styling := range p.categories.GetAll() {
		if styling.Cat.Deprecated {
			continue
		}
		box := util.Rect{
			X: x + p.horizPadding,
			Y: p.vertPadding + i + (i * p.gap),
			W: w - (2 * p.horizPadding),
			H: 1,
		}
		if styling.Cat.Name == p.currentCategory.Name && p.horizPadding > 0 {
			box.X -= 1
			box.W += 2
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
	return &ui.ToolsPanePositionInfo{Category: p.getCategoryForPos(x, y)}
}

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
	visible func() bool,
) *ToolsPane {
	return &ToolsPane{
		LeafPane: ui.LeafPane{
			BasePane: ui.BasePane{
				ID:             ui.GeneratePaneID(),
				InputProcessor: inputProcessor,
				Visible:        visible,
			},
			Renderer:   renderer,
			Dims:       dimensions,
			Stylesheet: stylesheet,
		},
		currentCategory: currentCategory,
		categories:      categories,
		horizPadding:    horizPadding,
		vertPadding:     vertPadding,
		gap:             gap,
	}
}
