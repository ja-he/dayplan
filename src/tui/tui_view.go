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
	panes := p.getCurrentlyActivePanesInOrder()
	lastIdx := len(panes) - 1

	// go through panes in reverse order (topmost drawn to bottommost drawn)
	for i := range panes {
		if util.NewRect(panes[lastIdx-i].Dimensions()).Contains(x, y) {
			return panes[lastIdx-i].GetPositionInfo(x, y)
		}
	}

	panic("argh!")
}

func (p *TUI) getCurrentlyActivePanesInOrder() []ui.UIPane {
	panes := make([]ui.UIPane, 0)

	switch *p.activeView {
	case ui.ViewDay:
		panes = append(panes, p.dayViewMainPane)
	case ui.ViewWeek:
		panes = append(panes, p.weekViewMainPane)
	case ui.ViewMonth:
		panes = append(panes, p.monthViewMainPane)
	}
	if p.editor.Condition() {
		panes = append(panes, p.editor)
	}
	if p.log.Condition() {
		panes = append(panes, p.log)
	}
	if p.summary.Condition() {
		panes = append(panes, p.summary)
	}
	if p.help.Condition() {
		panes = append(panes, p.help)
	}

	return panes
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

	panes := t.getCurrentlyActivePanesInOrder()
	for _, pane := range panes {
		pane.Draw()
	}

	t.renderer.Show()
}

type timestampStyle struct {
	timestamp model.Timestamp
	style     tcell.Style
}
