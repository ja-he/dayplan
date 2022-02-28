package tui

import (
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/ui"

	"github.com/gdamore/tcell/v2"
)

type TUI struct {
	renderer *TUIScreenHandler

	dimensions func() (x, y, w, h int)

	dayViewMainPane   ui.UIPane
	weekViewMainPane  ui.UIPane
	monthViewMainPane ui.UIPane

	summary ui.UIPane
	log     ui.UIPane

	help   ui.UIPane
	editor ui.UIPane

	activeView *ui.ActiveView
}

func (t *TUI) Dimensions() (x, y, w, h int) {
	return t.dimensions()
}

func (p *TUI) GetPositionInfo(x, y int) ui.PositionInfo {
	// TODO: other panes?

	return p.dayViewMainPane.GetPositionInfo(x, y)
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

	switch *t.activeView {
	case ui.ViewDay:
		t.dayViewMainPane.Draw()
	case ui.ViewWeek:
		t.weekViewMainPane.Draw()
	case ui.ViewMonth:
		t.monthViewMainPane.Draw()
	}
	t.editor.Draw()
	t.log.Draw()
	t.summary.Draw()
	t.help.Draw()

	t.renderer.Show()
}

type timestampStyle struct {
	timestamp model.Timestamp
	style     tcell.Style
}
