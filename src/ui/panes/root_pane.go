package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

// RootPane acts as the root UI pane, wrapping all subpanes, managing the
// render cycle, invoking the subpanes' rendering, etc.
type RootPane struct {
	ID ui.PaneID

	renderer ui.RenderOrchestratorControl

	dimensions func() (x, y, w, h int)

	focussedViewPane ui.Pane

	dayViewMainPane   ui.Pane
	weekViewMainPane  ui.Pane
	monthViewMainPane ui.Pane

	summary ui.Pane
	log     ui.Pane

	help   ui.Pane
	editor ui.Pane

	performanceMetricsOverlay ui.Pane

	inputProcessor input.ModalInputProcessor
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

func (p *RootPane) getCurrentlyActivePanesInOrder() (active []ui.Pane, inactive []ui.Pane) {
	active = make([]ui.Pane, 0)
	inactive = make([]ui.Pane, 0)

	// append day, week, or month pane
	active = append(active, p.focussedViewPane)

	// TODO: this change breaks the cursor hiding, as that is done in the draw
	//       call when !condition. it should be done differently anyways though,
	//       imo.
	if p.editor.IsVisible() {
		active = append(active, p.editor)
	} else {
		inactive = append(inactive, p.editor)
	}
	if p.log.IsVisible() {
		active = append(active, p.log)
	} else {
		inactive = append(inactive, p.log)
	}
	if p.summary.IsVisible() {
		active = append(active, p.summary)
	} else {
		inactive = append(inactive, p.summary)
	}
	if p.help.IsVisible() {
		active = append(active, p.help)
	} else {
		inactive = append(inactive, p.help)
	}

	return active, inactive
}

func (p *RootPane) IsVisible() bool { return true }

// Draw draws this pane.
func (p *RootPane) Draw() {
	p.renderer.Clear()

	active, inactive := p.getCurrentlyActivePanesInOrder()
	for _, pane := range active {
		pane.Draw()
	}
	for _, pane := range inactive {
		pane.Undraw()
	}

	p.performanceMetricsOverlay.Draw()

	p.renderer.Show()
}

func (p *RootPane) Undraw() {
	p.renderer.Clear()

	active, inactive := p.getCurrentlyActivePanesInOrder()
	for _, pane := range active {
		pane.Undraw()
	}
	for _, pane := range inactive {
		pane.Undraw()
	}

	p.performanceMetricsOverlay.Undraw()

	p.renderer.Show()
}

// CapturesInput returns whether this processor "captures" input, i.E. whether
// it ought to take priority in processing over other processors.
func (p *RootPane) CapturesInput() bool {
	if p.focussedPane().CapturesInput() {
		return true
	}
	return p.inputProcessor.CapturesInput()
}

// ProcessInput attempts to process the provided input.
// Returns whether the provided input "applied", i.E. the processor performed
// an action based on the input.
// Defers to the panes' input processor or its focussed subpanes.
func (p *RootPane) ProcessInput(key input.Key) bool {
	if p.inputProcessor.CapturesInput() {
		return p.inputProcessor.ProcessInput(key)
	} else if p.focussedPane().CapturesInput() {
		return p.focussedPane().ProcessInput(key)
	} else {
		return p.focussedPane().ProcessInput(key) || p.inputProcessor.ProcessInput(key)
	}
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

func (p *RootPane) Identify() ui.PaneID { return p.ID }
func (p *RootPane) HasFocus() bool      { return true }
func (p *RootPane) Focusses() ui.PaneID {
	return p.focussedPane().Identify()
}
func (p *RootPane) FocusPrev() {}
func (p *RootPane) FocusNext() {}

func (p *RootPane) focussedPane() ui.Pane {
	switch {
	case p.help.IsVisible():
		return p.help
	case p.editor.IsVisible():
		return p.editor
	case p.summary.IsVisible():
		return p.summary
	case p.log.IsVisible():
		return p.log
	default:
		return p.focussedViewPane
	}
}
func (p *RootPane) SetParent(ui.PaneQuerier) { panic("root set parent") }

// ApplyModalOverlay applies an overlay to this processor.
// It returns the processors index, by which in the future, all overlays down
// to and including this overlay can be removed
func (p *RootPane) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index uint) {
	return p.inputProcessor.ApplyModalOverlay(overlay)
}

// PopModalOverlay removes the topmost overlay from this processor.
func (p *RootPane) PopModalOverlay() error {
	return p.inputProcessor.PopModalOverlay()
}

// PopModalOverlays pops all overlays down to and including the one at the
// specified index.
func (p *RootPane) PopModalOverlays(index uint) {
	p.inputProcessor.PopModalOverlays(index)
}

// GetHelp returns the input help map for this processor.
func (p *RootPane) GetHelp() input.Help {
	result := input.Help{}

	for k, v := range p.inputProcessor.GetHelp() {
		result[k] = v
	}
	for k, v := range p.focussedPane().GetHelp() {
		result[k] = v
	}

	return result
}

// NewRootPane constructs and returns a new RootPane.
func NewRootPane(
	renderer ui.RenderOrchestratorControl,
	dimensions func() (x, y, w, h int),
	dayViewMainPane *Composite,
	weekViewMainPane *Composite,
	monthViewMainPane *Composite,
	summary ui.Pane,
	log ui.Pane,
	help ui.Pane,
	editor ui.Pane,
	performanceMetricsOverlay ui.Pane,
	inputProcessor input.ModalInputProcessor,
	focussedPane ui.Pane,
) *RootPane {
	rootPane := &RootPane{
		ID:                        ui.GeneratePaneID(),
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
		inputProcessor:            inputProcessor,
		focussedViewPane:          focussedPane,
	}
	dayViewMainPane.SetParent(rootPane)
	weekViewMainPane.SetParent(rootPane)
	monthViewMainPane.SetParent(rootPane)

	summary.SetParent(rootPane)
	help.SetParent(rootPane)
	editor.SetParent(rootPane)
	log.SetParent(rootPane)

	return rootPane
}
