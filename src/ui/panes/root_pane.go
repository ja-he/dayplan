package panes

import (
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

// RootPane acts as the root UI pane, wrapping all subpanes, managing the
// render cycle, invoking the subpanes' rendering, etc.
type RootPane struct {
	// TODO: I don't think I even want the root pane to handle the sync; in any
	//       case it shouldn't have to need this; remove.
	renderer ui.RootPaneRendererControl

	dimensions func() (x, y, w, h int)

	dayViewMainPane   ui.Pane
	weekViewMainPane  ui.Pane
	monthViewMainPane ui.Pane

	summary ui.ConditionalOverlayPane
	log     ui.ConditionalOverlayPane

	help   ui.ConditionalOverlayPane
	editor ui.ConditionalOverlayPane

	activeView func() *ui.ActiveView
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
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

	switch *p.activeView() {
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

// Close calls Fini() on the renderer, which, e.g., in the case of tcell.Screen
// finalizes the screen and releases all resources.
func (p *RootPane) Close() {
	p.renderer.Fini()
}

// NeedsSync informs the renderer of the need for sync, e.g., on a resize.
func (p *RootPane) NeedsSync() {
	p.renderer.NeedsSync()
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

	p.renderer.Show()
}

// NewRootPane constructs and returns a new RootPane.
func NewRootPane(
	renderer ui.RootPaneRendererControl,
	dimensions func() (x, y, w, h int),
	dayViewMainPane ui.Pane,
	weekViewMainPane ui.Pane,
	monthViewMainPane ui.Pane,
	summary ui.ConditionalOverlayPane,
	log ui.ConditionalOverlayPane,
	help ui.ConditionalOverlayPane,
	editor ui.ConditionalOverlayPane,
	activeView func() *ui.ActiveView,
) *RootPane {
	return &RootPane{
		renderer:          renderer,
		dimensions:        dimensions,
		dayViewMainPane:   dayViewMainPane,
		weekViewMainPane:  weekViewMainPane,
		monthViewMainPane: monthViewMainPane,
		summary:           summary,
		log:               log,
		help:              help,
		editor:            editor,
		activeView:        activeView,
	}
}
