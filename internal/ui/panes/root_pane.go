package panes

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
)

// RootPane acts as the root UI pane, wrapping all subpanes, managing the
// render cycle, invoking the subpanes' rendering, etc.
type RootPane struct {
	ID ui.PaneID

	renderer       ui.RenderOrchestratorControl
	cursorWrangler *ui.CursorWrangler

	dimensions func() (x, y, w, h int)

	focussedViewPane ui.Pane

	dayViewMainPane   ui.Pane
	weekViewMainPane  ui.Pane
	monthViewMainPane ui.Pane

	summaryPane ui.Pane
	logPane     ui.Pane

	helpPane ui.Pane

	subpanesMtx sync.Mutex
	subpanes    []ui.Pane

	performanceMetricsOverlay ui.Pane

	inputProcessor input.ModalInputProcessor

	preDrawStackMtx sync.Mutex
	preDrawStack    []func()

	log zerolog.Logger
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

	if p.summaryPane.IsVisible() {
		active = append(active, p.summaryPane)
	} else {
		inactive = append(inactive, p.summaryPane)
	}

	for i := range p.subpanes {
		if p.subpanes[i].IsVisible() {
			active = append(active, p.subpanes[i])
		} else {
			inactive = append(inactive, p.subpanes[i])
		}
	}

	if p.logPane.IsVisible() {
		active = append(active, p.logPane)
	} else {
		inactive = append(inactive, p.logPane)
	}

	// TODO: help should probably be a subpane? for now, always on top.
	if p.helpPane.IsVisible() {
		active = append(active, p.helpPane)
	} else {
		inactive = append(inactive, p.helpPane)
	}

	return active, inactive
}

func (p *RootPane) IsVisible() bool { return true }

// Draw draws this pane.
func (p *RootPane) Draw() {

	p.preDrawStackMtx.Lock()
	for _, f := range p.preDrawStack {
		f()
	}
	p.preDrawStack = nil
	p.preDrawStackMtx.Unlock()

	p.subpanesMtx.Lock()
	defer p.subpanesMtx.Unlock()

	p.renderer.Clear()

	// FIXME: probably simplify this
	active, _ := p.getCurrentlyActivePanesInOrder()
	for _, pane := range active {
		p.log.Trace().Msgf("drawing %d...", pane.Identify())
		pane.Draw()
		p.log.Trace().Msgf("drew %d.", pane.Identify())
	}
	// for _, pane := range _ {
	// 	pane.Undraw()
	// }

	p.performanceMetricsOverlay.Draw()

	// After all drawing draw or hide the cursor, depending on what is requested
	// during the draw of subpanes.
	p.cursorWrangler.Enact()

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

		processAttemptResult := p.focussedPane().ProcessInput(key)
		if processAttemptResult {
			return true
		}

		return p.inputProcessor.ProcessInput(key)

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
	case p.helpPane.IsVisible():
		return p.helpPane
	case p.summaryPane.IsVisible():
		return p.summaryPane
	case p.logPane.IsVisible():
		return p.logPane
	default:
		for i := range p.subpanes {
			if p.subpanes[i].IsVisible() {
				return p.subpanes[i]
			}
		}
		return p.focussedViewPane
	}
}
func (p *RootPane) SetParent(ui.PaneQuerier) { panic("root set parent") }

// DeferPreDraw attaches a function to the pre-draw stack, which is executed
func (p *RootPane) DeferPreDraw(f func()) {
	p.preDrawStackMtx.Lock()
	p.preDrawStack = append(p.preDrawStack, f)
	p.preDrawStackMtx.Unlock()
}

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

// PushSubpane allows adding a subpane over top of other subpanes.
func (p *RootPane) PushSubpane(pane ui.Pane) {
	pane.SetParent(p)
	p.subpanesMtx.Lock()
	defer p.subpanesMtx.Unlock()
	p.subpanes = append(p.subpanes, pane)
}

// PopSubpane pops the topmost subpane
func (p *RootPane) PopSubpane() {
	p.subpanesMtx.Lock()
	defer p.subpanesMtx.Unlock()
	if len(p.subpanes) == 0 {
		return
	}
	p.DeferPreDraw(p.subpanes[len(p.subpanes)-1].Undraw)
	p.subpanes = p.subpanes[:len(p.subpanes)-1]
}

// NewRootPane constructs and returns a new RootPane.
func NewRootPane(
	renderer ui.RenderOrchestratorControl,
	cursorWrangler *ui.CursorWrangler,
	dimensions func() (x, y, w, h int),
	dayViewMainPane *Composite,
	weekViewMainPane *Composite,
	monthViewMainPane *Composite,
	summaryPane ui.Pane,
	logPane ui.Pane,
	helpPane ui.Pane,
	performanceMetricsOverlay ui.Pane,
	inputProcessor input.ModalInputProcessor,
	focussedPane ui.Pane,
) *RootPane {
	rootPane := &RootPane{
		ID:                        ui.GeneratePaneID(),
		renderer:                  renderer,
		cursorWrangler:            cursorWrangler,
		dimensions:                dimensions,
		dayViewMainPane:           dayViewMainPane,
		weekViewMainPane:          weekViewMainPane,
		monthViewMainPane:         monthViewMainPane,
		summaryPane:               summaryPane,
		logPane:                   logPane,
		helpPane:                  helpPane,
		performanceMetricsOverlay: performanceMetricsOverlay,
		inputProcessor:            inputProcessor,
		focussedViewPane:          focussedPane,
		log:                       log.With().Str("component", "root-pane").Logger(),
	}
	defer rootPane.log.Trace().Msgf("created root pane with id '%d'", rootPane.Identify())

	dayViewMainPane.SetParent(rootPane)
	weekViewMainPane.SetParent(rootPane)
	monthViewMainPane.SetParent(rootPane)

	summaryPane.SetParent(rootPane)
	helpPane.SetParent(rootPane)
	logPane.SetParent(rootPane)

	return rootPane
}
