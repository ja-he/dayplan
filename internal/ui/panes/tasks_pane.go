package panes

import (
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
)

// TasksPane shows a tasks backlog from which tasks and prospective events can
// be selected and moved into concrete days, i.e., planned.
type TasksPane struct {
	Leaf
	backlog *model.Backlog
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

	// background
	func() {
		style := p.stylesheet.Normal
		if p.HasFocus() {
			style = p.stylesheet.NormalEmphasized
		}
		p.renderer.DrawBox(x, y, w, h, style)
	}()

	// title
	func() {
		style := p.stylesheet.NormalEmphasized.DefaultEmphasized()

		p.renderer.DrawBox(x, y, w, 1, style)

		titleText := "Backlog"
		p.renderer.DrawText(x+(w/2)-(len(titleText)/2), y, len(titleText), 1, style.Bolded(), titleText)
	}()

	// draws task, returns y space used
	drawTask := func(xBase, yOffset, wBase int, t model.Task) int {
		h := 2 // TODO: make based on duration and viewparams

		p.renderer.DrawBox(
			xBase+1, yOffset, wBase-2, h,
			p.stylesheet.CategoryFallback,
		)
		p.renderer.DrawText(
			xBase+1, yOffset, wBase-2, 1,
			p.stylesheet.CategoryFallback.Bolded(),
			util.TruncateAt(t.Name, wBase-2),
		)
		p.renderer.DrawText(
			xBase+3, yOffset+1, wBase-2-2, 1,
			p.stylesheet.CategoryFallback.Italicized(),
			util.TruncateAt(t.Category, wBase-2-2),
		)

		return h
	}

	// draw tasks
	func() {
		yIter := y
		for _, task := range p.backlog.Tasks {
			yIter += 1
			yIter += drawTask(x, yIter, w, task)
		}
	}()
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
	backlog *model.Backlog,
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
		backlog: backlog,
	}
}
