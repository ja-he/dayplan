package tui

import (
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"

	"github.com/gdamore/tcell/v2"
)

type TUI struct {
	// TODO: I don't think I even want the TUI to handle the sync; in any case it
	//       shouldn't have to need this; remove.
	renderer *TUIScreenHandler

	dimensions func() (x, y, w, h int)

	dayViewMainPane   ui.UIPane
	weekViewMainPane  ui.UIPane
	monthViewMainPane ui.UIPane

	summary ui.ConditionalOverlayPane
	log     ui.ConditionalOverlayPane

	help   ui.ConditionalOverlayPane
	editor ui.ConditionalOverlayPane

	activeView *ui.ActiveView
}

func (p *TUI) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *TUI) GetPositionInfo(x, y int) ui.PositionInfo {
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

func (p *TUI) getCurrentlyActivePanesInOrder() (active []ui.UIPane, inactive []ui.ConditionalOverlayPane) {
	active = make([]ui.UIPane, 0)
	inactive = make([]ui.ConditionalOverlayPane, 0)

	switch *p.activeView {
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

type TUIPositionInfo struct {
	paneType ui.UIPaneType
	weather  ui.WeatherPanePositionInfo
	timeline ui.TimelinePanePositionInfo
	tools    ui.ToolsPanePositionInfo
	status   ui.StatusPanePositionInfo
	events   ui.EventsPanePositionInfo
}

func (t *TUIPositionInfo) GetExtraWeatherInfo() ui.WeatherPanePositionInfo {
	return nil
}
func (t *TUIPositionInfo) GetExtraTimelineInfo() ui.TimelinePanePositionInfo {
	return nil
}
func (t *TUIPositionInfo) GetExtraToolsInfo() ui.ToolsPanePositionInfo {
	return t.tools
}
func (t *TUIPositionInfo) GetExtraStatusInfo() ui.StatusPanePositionInfo {
	return nil
}
func (t *TUIPositionInfo) GetExtraEventsInfo() ui.EventsPanePositionInfo {
	return t.events
}

func (t *TUIPositionInfo) PaneType() ui.UIPaneType {
	return t.paneType
}

func (p *TUI) Close() {
	p.renderer.Fini()
}

func (v *TUI) NeedsSync() {
	v.renderer.NeedsSync()
}

func (t *TUI) Draw() {
	t.renderer.Clear()

	active, inactive := t.getCurrentlyActivePanesInOrder()
	for _, pane := range active {
		pane.Draw()
	}
	for _, pane := range inactive {
		pane.EnsureHidden()
	}

	t.renderer.Show()
}

type timestampStyle struct {
	timestamp model.Timestamp
	style     tcell.Style
}
