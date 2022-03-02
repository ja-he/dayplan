package tui

import (
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"

	"github.com/gdamore/tcell/v2"
)

type TUI struct {
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

type TUIPositionTimestampGuessingWrapper struct {
	baseInfo       ui.PositionInfo
	timestampGuess model.Timestamp
}

func (t *TUIPositionTimestampGuessingWrapper) GetCursorTimestampGuess() (*model.Timestamp, error) {
	return &t.timestampGuess, nil
}
func (t *TUIPositionTimestampGuessingWrapper) GetExtraWeatherInfo() *ui.WeatherPanelPositionInfo {
	return t.baseInfo.GetExtraWeatherInfo()
}
func (t *TUIPositionTimestampGuessingWrapper) GetExtraTimelineInfo() *ui.TimelinePanelPositionInfo {
	return t.baseInfo.GetExtraTimelineInfo()
}
func (t *TUIPositionTimestampGuessingWrapper) GetExtraToolsInfo() *ui.ToolsPanelPositionInfo {
	return t.baseInfo.GetExtraToolsInfo()
}
func (t *TUIPositionTimestampGuessingWrapper) GetExtraStatusInfo() *ui.StatusPanelPositionInfo {
	return t.baseInfo.GetExtraStatusInfo()
}
func (t *TUIPositionTimestampGuessingWrapper) GetExtraEventsInfo() *ui.EventsPanelPositionInfo {
	return t.baseInfo.GetExtraEventsInfo()
}
func (t *TUIPositionTimestampGuessingWrapper) PaneType() ui.UIPaneType { return t.baseInfo.PaneType() }

func (t *TUIPositionInfo) GetCursorTimestampGuess() (*model.Timestamp, error) {
	// TODO: timestamp guess should not be valid, this should return error if
	//       e.g. the summary view is active
	return &t.timestampGuess, nil
}

type TUIPositionInfo struct {
	paneType       ui.UIPaneType
	weather        ui.WeatherPanelPositionInfo
	timeline       ui.TimelinePanelPositionInfo
	tools          ui.ToolsPanelPositionInfo
	status         ui.StatusPanelPositionInfo
	events         ui.EventsPanelPositionInfo
	timestampGuess model.Timestamp
}

func (t *TUIPositionInfo) GetExtraWeatherInfo() *ui.WeatherPanelPositionInfo {
	return &ui.WeatherPanelPositionInfo{}
}
func (t *TUIPositionInfo) GetExtraTimelineInfo() *ui.TimelinePanelPositionInfo {
	return &ui.TimelinePanelPositionInfo{}
}
func (t *TUIPositionInfo) GetExtraToolsInfo() *ui.ToolsPanelPositionInfo {
	return &t.tools
}
func (t *TUIPositionInfo) GetExtraStatusInfo() *ui.StatusPanelPositionInfo {
	return &ui.StatusPanelPositionInfo{}
}
func (t *TUIPositionInfo) GetExtraEventsInfo() *ui.EventsPanelPositionInfo {
	return &t.events
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
