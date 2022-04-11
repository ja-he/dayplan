package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

// RootPane acts as the root UI pane, wrapping all subpanes, managing the
// render cycle, invoking the subpanes' rendering, etc.
type RootPane struct {
	renderer ui.RenderOrchestratorControl

	dimensions func() (x, y, w, h int)

	focussedViewPane ui.FocussablePane

	dayViewMainPane   ui.FocussablePane
	weekViewMainPane  ui.FocussablePane
	monthViewMainPane ui.FocussablePane

	summary ui.ConditionalOverlayPane
	log     ui.ConditionalOverlayPane

	help   ui.ConditionalOverlayPane
	editor ui.ConditionalOverlayPane

	performanceMetricsOverlay ui.EphemeralPane

	inputTree input.Tree
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
func (p *RootPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *RootPane) GetPositionInfo(x, y int) ui.PositionInfo {
	activePanes, _ := p.getCurrentlyActivePanesInOrder()
	lastIdx := len(activePanes) - 1

	// go through panes in reverse order (topmost drawn to bottommost drawn)
	for i := range activePanes {
		if util.NewRect(activePanes[lastIdx-i].Dimensions()).Contains(x, y) {
			return activePanes[lastIdx-i].GetPositionInfo(x, y)
		}
	}

	panic("argh!")
}

func (p *RootPane) getCurrentlyActivePanesInOrder() (active []ui.Pane, inactive []ui.ConditionalOverlayPane) {
	active = make([]ui.Pane, 0)
	inactive = make([]ui.ConditionalOverlayPane, 0)

	// append day, week, or month pane
	active = append(active, p.focussedViewPane)

	// TODO: this change breaks the cursor hiding, as that is done in the draw
	//       call when !condition. it should be done differently anyways though,
	//       imo.
	if p.editor.Condition() {
		active = append(active, p.editor)
	} else {
		inactive = append(inactive, p.editor)
	}
	if p.log.Condition() {
		active = append(active, p.log)
	} else {
		inactive = append(inactive, p.log)
	}
	if p.summary.Condition() {
		active = append(active, p.summary)
	} else {
		inactive = append(inactive, p.summary)
	}
	if p.help.Condition() {
		active = append(active, p.help)
	} else {
		inactive = append(inactive, p.help)
	}

	return active, inactive
}

// Draw draws this pane.
func (p *RootPane) Draw() {
	p.renderer.Clear()

	active, inactive := p.getCurrentlyActivePanesInOrder()
	for _, pane := range active {
		pane.Draw()
	}
	for _, pane := range inactive {
		pane.EnsureHidden()
	}

	p.performanceMetricsOverlay.Draw()

	p.renderer.Show()
}

func (p *RootPane) HasPartialInput() bool {
	if p.Focusses().HasPartialInput() {
		return true
	}
	return p.inputTree.Active()
}
func (p *RootPane) ProcessInput(key input.Key) bool {
	// if own input tree active, capture processing
	if p.inputTree.Active() {
		return p.inputTree.Process(key)
	}

	// if input applies to focussed processor, let it process
	if p.Focusses().ProcessInput(key) {
		return true
	}

	// otherwise, process yourself
	return p.inputTree.Process(key)
}

func (p *RootPane) ViewUp() {
	switch p.focussedViewPane {
	case p.dayViewMainPane:
		p.focussedViewPane = p.weekViewMainPane
	case p.weekViewMainPane:
		p.focussedViewPane = p.monthViewMainPane
	case p.monthViewMainPane:
		return
	default:
		panic("unknown focussed pane")
	}
}

func (p *RootPane) ViewDown() {
	switch p.focussedViewPane {
	case p.dayViewMainPane:
		return
	case p.weekViewMainPane:
		p.focussedViewPane = p.dayViewMainPane
	case p.monthViewMainPane:
		p.focussedViewPane = p.weekViewMainPane
	default:
		panic("unknown focussed pane")
	}
}

func (p *RootPane) GetView() ui.ActiveView {
	switch p.focussedViewPane {
	case p.dayViewMainPane:
		return ui.ViewDay
	case p.weekViewMainPane:
		return ui.ViewWeek
	case p.monthViewMainPane:
		return ui.ViewMonth
	default:
		panic("unknown focussed pane")
	}
}

func (p *RootPane) HasFocus() bool              { return true }
func (p *RootPane) Focusses() ui.FocussablePane { return p.focussedViewPane }

func (p *RootPane) ApplyModalOverlay(input.Tree) (index int) {
	panic("todo: RootPane::ApplyModalOverlay")
}
func (p *RootPane) PopModalOverlay()           { panic("todo: RootPane::PopModalOverlay") }
func (p *RootPane) PopModalOverlays(index int) { panic("todo: RootPane::PopModalOverlays") }

// NewRootPane constructs and returns a new RootPane.
func NewRootPane(
	renderer ui.RenderOrchestratorControl,
	dimensions func() (x, y, w, h int),
	dayViewMainPane *DayViewMainPane,
	weekViewMainPane *WeekViewMainPane,
	monthViewMainPane *MonthViewMainPane,
	summary ui.ConditionalOverlayPane,
	log ui.ConditionalOverlayPane,
	help ui.ConditionalOverlayPane,
	editor ui.ConditionalOverlayPane,
	performanceMetricsOverlay ui.EphemeralPane,
	inputTree input.Tree,
	focussedPane ui.FocussablePane,
) *RootPane {
	rootPane := &RootPane{
		renderer:                  renderer,
		dimensions:                dimensions,
		dayViewMainPane:           dayViewMainPane,
		weekViewMainPane:          weekViewMainPane,
		monthViewMainPane:         monthViewMainPane,
		summary:                   summary,
		log:                       log,
		help:                      help,
		editor:                    editor,
		performanceMetricsOverlay: performanceMetricsOverlay,
		inputTree:                 inputTree,
		focussedViewPane:          focussedPane,
	}
	dayViewMainPane.Parent = rootPane
	weekViewMainPane.Parent = rootPane
	monthViewMainPane.Parent = rootPane
	return rootPane
}
