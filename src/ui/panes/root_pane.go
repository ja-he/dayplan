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

	dayViewMainPane interface {
		ui.Pane
		input.InputProcessor
	}
	weekViewMainPane  ui.Pane
	monthViewMainPane ui.Pane

	summary ui.ConditionalOverlayPane
	log     ui.ConditionalOverlayPane

	help   ui.ConditionalOverlayPane
	editor ui.ConditionalOverlayPane

	performanceMetricsOverlay ui.EphemeralPane

	inputTree input.Tree

	activeView func() ui.ActiveView
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

	switch p.activeView() {
	case ui.ViewDay:
		active = append(active, p.dayViewMainPane)
	case ui.ViewWeek:
		active = append(active, p.weekViewMainPane)
	case ui.ViewMonth:
		active = append(active, p.monthViewMainPane)
	}
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

func (p *RootPane) inputProcessors() []input.InputProcessor {
	result := make([]input.InputProcessor, 0)

	switch {
	case p.activeView() == ui.ViewDay && !p.log.Condition() && !p.summary.Condition() && !p.help.Condition() && !p.editor.Condition():
		result = append(result, p.dayViewMainPane)
	}

	return result
}
func (p *RootPane) HasPartialInput() bool {
	for _, processor := range p.inputProcessors() {
		if processor.HasPartialInput() {
			return true
		}
	}
	return p.inputTree.Active()
}
func (p *RootPane) ProcessInput(key input.Key) bool {
	for _, processor := range p.inputProcessors() {
		if processor.ProcessInput(key) {
			return true
		}
	}
	return p.inputTree.Process(key)
}

// NewRootPane constructs and returns a new RootPane.
func NewRootPane(
	renderer ui.RenderOrchestratorControl,
	dimensions func() (x, y, w, h int),
	dayViewMainPane interface {
		ui.Pane
		input.InputProcessor
	},
	weekViewMainPane ui.Pane,
	monthViewMainPane ui.Pane,
	summary ui.ConditionalOverlayPane,
	log ui.ConditionalOverlayPane,
	help ui.ConditionalOverlayPane,
	editor ui.ConditionalOverlayPane,
	performanceMetricsOverlay ui.EphemeralPane,
	inputTree input.Tree,
	activeView func() ui.ActiveView,
) *RootPane {
	return &RootPane{
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
		activeView:                activeView,
	}
}
