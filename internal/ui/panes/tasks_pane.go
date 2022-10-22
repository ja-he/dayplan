package panes

import (
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
)

// TasksPane shows a tasks backlog from which tasks and prospective events can
// be selected and moved into concrete days, i.e., planned.
type TasksPane struct {
	Leaf
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *TasksPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw draws this pane.
func (p *TasksPane) Draw() {
	if !p.IsVisible() {
		return
	}

	x, y, w, h := p.dimensions()

	style := p.stylesheet.CategoryFallback
	if p.HasFocus() {
		style = p.stylesheet.CategoryFallback.DarkenedBG(20)
	}
	p.renderer.DrawBox(x, y, w, h, style)
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *TasksPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return &TasksPanePositionInfo{}
}

// TasksPanePositionInfo conveys information on a position in a tasks pane.
type TasksPanePositionInfo struct {
}

// NewTasksPane constructs and returns a new TasksPane.
func NewTasksPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	inputProcessor input.ModalInputProcessor,
	visible func() bool,
) *TasksPane {
	return &TasksPane{
		Leaf: Leaf{
			Base: Base{
				ID:             ui.GeneratePaneID(),
				InputProcessor: inputProcessor,
				Visible:        visible,
			},
			renderer:   renderer,
			dimensions: dimensions,
			stylesheet: stylesheet,
		},
	}
}
