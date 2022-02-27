package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/colors"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"

	"github.com/gdamore/tcell/v2"
)

type TUI struct {
	renderer *TUIRenderer

	dimensions func() (x, y, w, h int)

	dayViewMainPane   ui.UIPane
	weekViewMainPane  ui.UIPane
	monthViewMainPane ui.UIPane

	// TODO: split up, probably
	days            *DaysData
	currentDate     *model.Date
	currentCategory *model.Category
	categories      *category_style.CategoryStyling
	eventEditor     *EventEditor
	showHelp        *bool
	showSummary     *bool
	showLog         *bool
	activeView      *ui.ActiveView
	logReader       potatolog.LogReader
	logWriter       potatolog.LogWriter
	viewParams      *ViewParams
	cursor          *CursorPos
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

const editorWidth = 80
const editorHeight = 20

const helpWidth = 80
const helpHeight = 30

func (t *TUI) getScreenCenter() (int, int) {
	w, h := t.renderer.GetScreenDimensions()
	x := w / 2
	y := h / 2
	return x, y
}

func (t *TUI) drawEditor() {
	editor := t.eventEditor
	style := tcell.StyleDefault.Background(tcell.ColorLightGrey).Foreground(tcell.ColorBlack)
	if editor.Active {
		x, y := t.getScreenCenter()
		x -= editorWidth / 2
		y -= editorHeight / 2
		t.renderer.DrawBox(style, x, y, editorWidth, editorHeight)
		t.renderer.DrawText(x+1, y+1, editorWidth-2, editorHeight-2, style, editor.TmpEventInfo.Name)
		t.renderer.ShowCursor(x+1+(editor.CursorPos%(editorWidth-2)), y+1+(editor.CursorPos/(editorWidth-2)))
		// TODO(ja-he): wrap at word boundary
	} else {
		t.renderer.HideCursor()
	}
}

// Draw the help popup.
func (t *TUI) drawHelp() {
	if *t.showHelp {

		helpStyle := tcell.StyleDefault.Background(tcell.ColorLightGrey)
		keyStyle := colors.DefaultEmphasize(helpStyle).Bold(true)
		descriptionStyle := helpStyle.Italic(true)

		x, y := t.getScreenCenter()
		x -= helpWidth / 2
		y -= helpHeight / 2
		t.renderer.DrawBox(helpStyle, x, y, helpWidth, helpHeight)

		keysDrawn := 0
		const border = 1
		const maxKeyWidth = 20
		const pad = 1
		keyOffset := x + border
		descriptionOffset := keyOffset + maxKeyWidth + pad

		drawMapping := func(keys, description string) {
			t.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keys)), y+border+keysDrawn, len([]rune(keys)), 0, keyStyle, keys)
			t.renderer.DrawText(descriptionOffset, y+border+keysDrawn, helpWidth, helpHeight, descriptionStyle, description)
			keysDrawn++
		}

		drawOpposedMapping := func(keyA, keyB, description string) {
			sepText := "/"
			t.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keyB))-len(sepText)-len([]rune(keyA)), y+border+keysDrawn, len([]rune(keyA)), 0, keyStyle, keyA)
			t.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keyB))-len(sepText), y+border+keysDrawn, len(sepText), 0, helpStyle, sepText)
			t.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keyB)), y+border+keysDrawn, len([]rune(keyB)), 0, keyStyle, keyB)
			t.renderer.DrawText(descriptionOffset, y+border+keysDrawn, helpWidth, helpHeight, descriptionStyle, description)
			keysDrawn++
		}

		space := func() { drawMapping("", "") }

		drawMapping("?", "toggle help")
		space()

		drawMapping("<lmb>[+<move down>]", "create or edit event")
		drawMapping("<rmb>", "split event (in event view)")
		drawMapping("<mmb>", "delete event")
		drawMapping("<ctrl-lmb>+<move>", "move event with following")
		space()

		drawOpposedMapping("<c-u>", "<c-d>", "scroll up / down")
		drawOpposedMapping("k", "j", "scroll up / down")
		drawOpposedMapping("g", "G", "scroll to top / bottom")
		space()

		drawOpposedMapping("+", "-", "zoom in / out")
		space()

		drawOpposedMapping("h", "l", "go to previous / next day")
		space()

		drawOpposedMapping("i", "<esc>", "narrow / broaden view")
		space()

		drawMapping("w", "write day to file")
		drawMapping("c", "clear day (remove all events)")
		drawMapping("q", "quit (unwritten data is lost)")
		space()

		drawMapping("S", "toggle summary")
		drawMapping("E", "toggle debug log")
		space()

		drawMapping("u", "update weather (requires some envvars)")
		space()
	}
}

// Draws the time summary view over top of all previously drawn contents, if it
// is currently active.
func (t *TUI) drawSummary() {
	style := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	if *t.showSummary {
		y := 2
		_, _, w, h := t.Dimensions()

		t.renderer.DrawBox(style, 0, 0, w, h)
		dateString := ""
		switch *t.activeView {
		case ui.ViewDay:
			dateString = t.currentDate.ToString()
		case ui.ViewWeek:
			start, end := t.currentDate.Week()
			dateString = fmt.Sprintf("week %s..%s", start.ToString(), end.ToString())
		case ui.ViewMonth:
			dateString = fmt.Sprintf("%s %d", t.currentDate.ToGotime().Month().String(), t.currentDate.Year)
		}
		title := fmt.Sprintf("SUMMARY (%s)", dateString)
		t.renderer.DrawBox(style.Background(tcell.ColorLightGrey), 0, 0, w, 1)
		t.renderer.DrawText(w/2-len(title)/2, 0, len(title), 1, style.Background(tcell.ColorLightGrey).Bold(true), title)

		summary := make(map[model.Category]int)
		switch *t.activeView {
		case ui.ViewDay:
			day := t.days.GetDay(*t.currentDate)
			if day == nil {
				return
			}
			summary = day.SumUpByCategory()
		case ui.ViewWeek:
			start, end := t.currentDate.Week()
			for current := start; current != end.Next(); current = current.Next() {
				day := t.days.GetDay(current)
				if day == nil {
					return
				}
				tmpSummary := day.SumUpByCategory()
				for k, v := range tmpSummary {
					summary[k] += v
				}
			}
		case ui.ViewMonth:
			start, end := t.currentDate.MonthBounds()
			for current := start; current != end.Next(); current = current.Next() {
				day := t.days.GetDay(current)
				if day == nil {
					return
				}
				tmpSummary := day.SumUpByCategory()
				for k, v := range tmpSummary {
					summary[k] += v
				}
			}
		}
		maxDuration := 0
		categories := make([]model.Category, len(summary))
		{ // get sorted keys to have deterministic order
			i := 0
			for category, duration := range summary {
				categories[i] = category
				if duration > maxDuration {
					maxDuration = duration
				}
				i++
			}
			sort.Sort(model.ByName(categories))
		}
		for _, category := range categories {
			duration := summary[category]
			style, _ := t.categories.GetStyle(category)
			catLen := 20
			durationLen := 20
			barWidth := int(float64(duration) / float64(maxDuration) * float64(w-catLen-durationLen))
			t.renderer.DrawBox(style, catLen+durationLen, y, barWidth, 1)
			t.renderer.DrawText(0, y, catLen, 0, tcell.StyleDefault, util.TruncateAt(category.Name, catLen))
			t.renderer.DrawText(catLen, y, durationLen, 0, style, "("+util.DurationToString(duration)+")")
			y++
		}
	}
}

func (t *TUI) drawLog() {
	style := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	if *t.showLog {
		_, _, w, h := t.Dimensions()
		x, y := 0, 2

		t.renderer.DrawBox(style, 0, 0, w, h)
		title := "LOG"
		t.renderer.DrawBox(style.Background(tcell.ColorLightGrey), 0, 0, w, 1)
		t.renderer.DrawText(w/2-len(title)/2, 0, len(title), 1, style.Background(tcell.ColorLightGrey).Bold(true), title)
		for i := len(t.logReader.Get()) - 1; i >= 0; i-- {
			entry := &t.logReader.Get()[i]

			t.renderer.DrawText(x, y, w, 0, style.Foreground(tcell.ColorDarkGrey).Italic(true), entry.Type)
			x += len(entry.Type) + 1

			t.renderer.DrawText(x, y, w, 0, style, entry.Message)
			x += len(entry.Message) + 1

			t.renderer.DrawText(x, y, w, 0, style.Foreground(tcell.ColorDarkGrey), entry.Location)
			x += len(entry.Location) + 1

			timeStr := strings.Join(strings.Split(entry.At.String(), " ")[0:2], " ")
			t.renderer.DrawText(x, y, w, 0, style.Foreground(tcell.ColorLightGrey), timeStr)

			x = 0
			y++
		}
	}
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
	t.drawEditor()
	t.drawLog()
	t.drawSummary()
	t.drawHelp()

	t.renderer.Show()
}

type timestampStyle struct {
	timestamp model.Timestamp
	style     tcell.Style
}
