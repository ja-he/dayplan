package panes

import (
	"sync"

	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
	"github.com/rs/zerolog/log"
)

// BacklogPane shows a tasks backlog from which tasks and prospective events can
// be selected and moved into concrete days, i.e., planned.
type BacklogPane struct {
	Leaf
	viewParams            ui.TimeViewParams
	getCurrentTask        func() *model.Task
	backlog               *model.Backlog
	categoryStyleProvider func(model.Category) (styling.DrawStyling, error)

	uiBoundsMtx sync.RWMutex
	uiBounds    map[*model.Task]taskUIYBounds
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *BacklogPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw draws this pane.
func (p *BacklogPane) Draw() {
	p.uiBoundsMtx.Lock()
	defer p.uiBoundsMtx.Unlock()

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

	// draws task, taking into account view params, returns y space used
	var drawTask func(xBase, yOffset, wBase int, t *model.Task, depth int, emphasize bool) (int, []func())
	drawTask = func(xBase, yOffset, wBase int, t *model.Task, depth int, emphasize bool) (int, []func()) {
		drawThis := []func(){}

		var h int
		if t.Duration == nil {
			h = 2 * int(p.viewParams.GetZoomPercentage()/50.0)
		} else {
			h = int(p.viewParams.HeightOfDuration(*t.Duration))
		}
		if len(t.Subtasks) > 0 {
			yIter := yOffset + 1
			for i, st := range t.Subtasks {
				drawnHeight, drawCalls := drawTask(xBase+1, yIter, wBase-2, st, depth+1, emphasize || p.getCurrentTask() == st)
				drawThis = append(drawThis, drawCalls...)
				effectiveYIncrease := drawnHeight
				if i != len(t.Subtasks)-1 {
					effectiveYIncrease += 1
				}
				h += effectiveYIncrease
				yIter += effectiveYIncrease
			}
		}

		style, err := p.categoryStyleProvider(t.Category)
		if err != nil {
			style = p.stylesheet.CategoryFallback
		}
		style = style.DarkenedBG(depth * 10)

		if emphasize {
			style = style.DefaultEmphasized()
		}

		if emphasize {
			xBase -= 1
			wBase += 2
		}
		drawThis = append(drawThis, func() {
			p.renderer.DrawBox(
				xBase+1, yOffset, wBase-2, h,
				style,
			)
			p.renderer.DrawText(
				xBase+1+1, yOffset, wBase-2-1, 1,
				style.Bolded(),
				util.TruncateAt(t.Name, wBase-2-1),
			)
			p.renderer.DrawText(
				xBase+3, yOffset+1, wBase-2-2, 1,
				style.Italicized(),
				util.TruncateAt(t.Category.Name, wBase-2-2),
			)
			if t.Deadline != nil {
				deadline := t.Deadline.Format("2006-01-02 15:04:05")
				p.renderer.DrawText(
					xBase+wBase-len(deadline)-1, yOffset+1, len(deadline), 1,
					style.Bolded(),
					deadline,
				)
			}
			p.uiBounds[t] = taskUIYBounds{yOffset, yOffset + h - 1}
		})

		return h, drawThis
	}

	// draw tasks
	func() {
		p.backlog.Mtx.RLock()
		defer p.backlog.Mtx.RUnlock()

		yIter := y + 1 - p.viewParams.GetScrollOffset()
		for _, task := range p.backlog.Tasks {
			yIter += 1
			heightDrawn, drawFuncs := drawTask(x+1, yIter, w-2, task, 0, p.getCurrentTask() == task)
			for i := range drawFuncs {
				drawFuncs[len(drawFuncs)-1-i]()
			}
			yIter += heightDrawn
		}
	}()

	// draw title last
	func() {
		style := p.stylesheet.NormalEmphasized.DefaultEmphasized()

		p.renderer.DrawBox(x, y, w, 1, style)

		titleText := "Backlog"
		p.renderer.DrawText(x+(w/2)-(len(titleText)/2), y, len(titleText), 1, style.Bolded(), titleText)
	}()
}

func (p *BacklogPane) GetTaskUIYBounds(t *model.Task) (lb, ub int) {
	p.uiBoundsMtx.RLock()
	defer p.uiBoundsMtx.RUnlock()
	r, ok := p.uiBounds[t]
	if ok {
		return r.lb, r.ub
	} else {
		log.Warn().Interface("task", t).
			Msg("backlog pane asked for position of unknown task")
		return 0, 0
	}
}

func (p *BacklogPane) GetTaskVisibilityBounds() (lb, ub int) {
	_, y, _, h := p.dimensions()
	return y, y + h - 1
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *BacklogPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return &BacklogPanePositionInfo{}
}

// BacklogPanePositionInfo conveys information on a position in a BacklogPane.
type BacklogPanePositionInfo struct {
}

// NewBacklogPane constructs and returns a new BacklogPane.
func NewBacklogPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	inputProcessor input.ModalInputProcessor,
	viewParams ui.TimeViewParams,
	getCurrentTask func() *model.Task,
	backlog *model.Backlog,
	categoryStyleProvider func(model.Category) (styling.DrawStyling, error),
	visible func() bool,
) *BacklogPane {
	p := BacklogPane{
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
		viewParams:            viewParams,
		getCurrentTask:        getCurrentTask,
		backlog:               backlog,
		categoryStyleProvider: categoryStyleProvider,
		uiBounds:              make(map[*model.Task]taskUIYBounds),
	}
	return &p
}

type taskUIYBounds struct {
	lb, ub int
}
