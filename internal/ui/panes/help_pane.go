package panes

import (
	"sort"

	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
)

// A HelpPane is a pane that displays a help popup.
// For example, it could display a list of key mappings and their actions.
type HelpPane struct {
	ui.LeafPane

	Content input.Help
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *HelpPane) Dimensions() (x, y, w, h int) {
	return p.Dims()
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *HelpPane) GetPositionInfo(x, y int) ui.PositionInfo { return nil }

// Draw draws the help popup.
func (p *HelpPane) Draw() {
	if p.IsVisible() {

		x, y, w, h := p.Dimensions()
		p.Renderer.DrawBox(x, y, w, h, p.Stylesheet.Help)

		keysDrawn := 0
		const border = 1
		const maxKeyWidth = 20
		const pad = 1
		keyOffset := x + border
		descriptionOffset := keyOffset + maxKeyWidth + pad

		drawMapping := func(keys, description string) {
			p.Renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keys)), y+border+keysDrawn, len([]rune(keys)), 1, p.Stylesheet.Help.DefaultEmphasized().Bolded(), keys)
			p.Renderer.DrawText(descriptionOffset, y+border+keysDrawn, w, h, p.Stylesheet.Help.Italicized(), description)
			keysDrawn++
		}

		content := make([]mappingAndAction, len(p.Content))
		{
			i := 0
			for mapping, action := range p.Content {
				content[i] = mappingAndAction{mapping: mapping, action: action}
				i++
			}
			sort.Sort(byAction(content))
		}
		for i := range content {
			drawMapping(content[i].mapping, content[i].action)
		}

	}
}

type mappingAndAction = struct {
	mapping string
	action  string
}
type byAction []mappingAndAction

func (a byAction) Len() int           { return len(a) }
func (a byAction) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byAction) Less(i, j int) bool { return a[i].action < a[j].action }

// NewHelpPane constructs and returns a new HelpPane.
func NewHelpPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	condition func() bool,
	inputProcessor input.ModalInputProcessor,
) *HelpPane {
	p := &HelpPane{
		LeafPane: ui.LeafPane{
			BasePane: ui.BasePane{
				ID:      ui.GeneratePaneID(),
				Visible: condition,
			},
			Renderer:   renderer,
			Dims:       dimensions,
			Stylesheet: stylesheet,
		},
	}
	p.InputProcessor = inputProcessor
	return p
}
