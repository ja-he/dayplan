package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/colors"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
	"github.com/ja-he/dayplan/src/weather"

	"github.com/gdamore/tcell/v2"
)

type TUI struct {
	renderer *TUIRenderer

	dimensions func() (x, y, w, h int)

	// get rid of -> closure
	uiDimensions UIDims

	tools  ui.UIPane
	status ui.UIPane

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
	weather         *weather.Handler
	viewParams      *ViewParams
	cursor          *CursorPos

	// TODO: get rid of this
	positions map[model.EventID]util.Rect
}

func (p *TUI) GetPositionInfo(x, y int) ui.PositionInfo {
	paneType := p.uiDimensions.WhichUIPane(x, y)

	switch paneType {
	case ui.ToolsUIPanelType:
		temp := p.tools.GetPositionInfo(x, y)
		return &TUIPositionTimestampGuessingWrapper{
			baseInfo:       temp,
			timestampGuess: p.TimeAtY(y),
		}
	default:
		return &TUIPositionInfo{
			paneType:       paneType,
			weather:        ui.WeatherPanelPositionInfo{},
			timeline:       ui.TimelinePanelPositionInfo{},
			tools:          ui.ToolsPanelPositionInfo{},
			status:         ui.StatusPanelPositionInfo{},
			events:         p.getEventForPos(x, y),
			timestampGuess: p.TimeAtY(y),
		}
	}
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

func (t *TUI) getEventForPos(x, y int) ui.EventsPanelPositionInfo {
	if x >= t.uiDimensions.EventsOffset() &&
		x < (t.uiDimensions.EventsOffset()+t.uiDimensions.EventsWidth()) {
		currentDay := t.days.GetDay(*t.currentDate)
		for i := len(currentDay.Events) - 1; i >= 0; i-- {
			eventPos := t.positions[currentDay.Events[i].ID]
			if eventPos.Contains(x, y) {
				var hover ui.EventHoverState
				switch {
				case y == (eventPos.Y+eventPos.H-1) && x > eventPos.X+eventPos.W-5:
					hover = ui.EventHoverStateResize
				case y == (eventPos.Y):
					hover = ui.EventHoverStateEdit
				default:
					hover = ui.EventHoverStateMove
				}
				return ui.EventsPanelPositionInfo{
					Event:           currentDay.Events[i].ID,
					HoverState:      hover,
					TimeUnderCursor: t.TimeAtY(y),
				}
			}
		}
	}
	return ui.EventsPanelPositionInfo{
		Event:           0,
		HoverState:      ui.EventHoverStateNone,
		TimeUnderCursor: t.TimeAtY(y),
	}
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
		y, w, h := 2, t.uiDimensions.screenWidth, t.uiDimensions.screenHeight
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
			barWidth := int(float64(duration) / float64(maxDuration) * float64(t.uiDimensions.screenWidth-catLen-durationLen))
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
		x, y, w, h := 0, 2, t.uiDimensions.screenWidth, t.uiDimensions.screenHeight
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

	// TODO: define all styles here (prep to probably move out further)
	headerBG := tcell.StyleDefault.Background(colors.ColorFromHexString("#f0f0f0")).Foreground(tcell.ColorBlack)
	headerBGEmph := colors.DefaultEmphasize(headerBG)
	dayBG := tcell.StyleDefault
	dayBGEmph := headerBGEmph
	loadingStyle := dayBG.Foreground(tcell.ColorLightSeaGreen)

	switch *t.activeView {
	case ui.ViewDay:
		t.drawWeather()
		t.drawTimeline()
		t.drawEvents()
		t.tools.Draw()
		t.drawEditor()
	case ui.ViewWeek:
		start, end := t.currentDate.Week()
		nDays := start.DaysUntil(end) + 1
		if nDays > t.uiDimensions.screenWidth {
			t.renderer.DrawText(0, 0, t.uiDimensions.screenWidth, t.uiDimensions.screenHeight,
				tcell.StyleDefault.Foreground(tcell.ColorRebeccaPurple),
				"refusing to render week on screen with fewer columns than days")
			return
		}

		{
			firstDayXOffset := 10
			x := firstDayXOffset
			dayWidth := (t.uiDimensions.screenWidth - firstDayXOffset) / nDays

			t.drawTimelineTmp(0, 0, firstDayXOffset, t.uiDimensions.screenHeight-t.uiDimensions.statusHeight, make([]timestampStyle, 0), nil)

			for drawDate := start; drawDate != end.Next(); drawDate = drawDate.Next() {
				if drawDate == *t.currentDate {
					t.renderer.DrawBox(dayBGEmph, x, 0, dayWidth, t.uiDimensions.screenHeight)
				} else {
					t.renderer.DrawBox(dayBG, x, 0, dayWidth, t.uiDimensions.screenHeight)
				}
				day := t.days.GetDay(drawDate)
				if day != nil {
					positions := t.ComputeRects(day, x, 0, dayWidth, t.uiDimensions.screenHeight-t.uiDimensions.statusHeight)
					for _, e := range day.Events {
						p := positions[e.ID]
						style, err := t.categories.GetStyle(e.Cat)
						if err != nil {
							panic(err)
						}
						if drawDate != *t.currentDate {
							style = colors.DefaultDim(style)
						}
						t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H)
						t.renderer.DrawText(p.X, p.Y, p.W, 0, style, util.TruncateAt(e.Name, p.W))
					}
				} else {
					loadingText := "⋮"
					t.renderer.DrawText(x, t.uiDimensions.screenHeight/2-len([]rune(loadingText)), 1, len([]rune(loadingText)),
						loadingStyle,
						loadingText)
				}
				x += dayWidth
			}
		}

	case ui.ViewMonth:
		start, end := t.currentDate.MonthBounds()
		nDays := start.DaysUntil(end) + 1
		if nDays > t.uiDimensions.screenWidth {
			t.renderer.DrawText(0, 0, t.uiDimensions.screenWidth, t.uiDimensions.screenHeight,
				tcell.StyleDefault.Foreground(tcell.ColorRebeccaPurple),
				"refusing to render month on screen with fewer columns than days")
			return
		}

		{
			firstDayXOffset := 10
			x := firstDayXOffset
			dayWidth := (t.uiDimensions.screenWidth - firstDayXOffset) / nDays

			t.drawTimelineTmp(0, 0, firstDayXOffset, t.uiDimensions.screenHeight-t.uiDimensions.statusHeight, make([]timestampStyle, 0), nil)

			for drawDate := start; drawDate != end.Next(); drawDate = drawDate.Next() {
				if drawDate == *t.currentDate {
					t.renderer.DrawBox(dayBGEmph, x, 0, dayWidth, t.uiDimensions.screenHeight)
				} else {
					t.renderer.DrawBox(dayBG, x, 0, dayWidth, t.uiDimensions.screenHeight)
				}
				day := t.days.GetDay(drawDate)
				if day != nil {
					positions := t.ComputeRects(day, x, 0, dayWidth, t.uiDimensions.screenHeight-t.uiDimensions.statusHeight)
					for _, e := range day.Events {
						p := positions[e.ID]
						style, err := t.categories.GetStyle(e.Cat)
						if err != nil {
							panic(err)
						}
						if drawDate != *t.currentDate {
							style = colors.DefaultDim(style)
						}
						t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H)
					}
				} else {
					loadingText := "⋮"
					t.renderer.DrawText(x, t.uiDimensions.screenHeight/2-len([]rune(loadingText)), 1, len([]rune(loadingText)),
						loadingStyle,
						loadingText)
				}
				x += dayWidth
			}
		}
	}
	t.status.Draw()
	t.drawLog()
	t.drawSummary()
	t.drawHelp()

	t.renderer.Show()
}

func (t *TUI) drawWeather() {
	for timestamp := *model.NewTimestamp("00:00"); timestamp.Legal(); timestamp.Hour++ {
		y := t.toY(timestamp)

		index := model.DayAndTime{
			Date:      *t.currentDate,
			Timestamp: timestamp,
		}

		weather, ok := t.weather.Data[index]
		if ok {
			weatherStyle := tcell.StyleDefault.Foreground(tcell.ColorLightBlue)
			switch {
			case weather.PrecipitationProbability > .25:
				weatherStyle = weatherStyle.Background(tcell.NewHexColor(0xccebff)).Foreground(tcell.ColorBlack)
			case weather.Clouds < 25:
				weatherStyle = weatherStyle.Background(tcell.NewHexColor(0xfff0cc)).Foreground(tcell.ColorBlack)
			}

			t.renderer.DrawBox(weatherStyle, t.uiDimensions.WeatherOffset(), y, t.uiDimensions.WeatherWidth(), t.viewParams.NRowsPerHour)

			t.renderer.DrawText(t.uiDimensions.WeatherOffset(), y, t.uiDimensions.WeatherWidth(), 0, weatherStyle, weather.Info)
			t.renderer.DrawText(t.uiDimensions.WeatherOffset(), y+1, t.uiDimensions.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%2.0f°C", weather.TempC))
			t.renderer.DrawText(t.uiDimensions.WeatherOffset(), y+2, t.uiDimensions.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%d%% clouds", weather.Clouds))
			t.renderer.DrawText(t.uiDimensions.WeatherOffset(), y+3, t.uiDimensions.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%d%% humidity", weather.Humidity))
			t.renderer.DrawText(t.uiDimensions.WeatherOffset(), y+4, t.uiDimensions.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%2.0f%% chance of rain", 100.0*weather.PrecipitationProbability))
		}
	}
}

type timestampStyle struct {
	timestamp model.Timestamp
	style     tcell.Style
}

func (t *TUI) drawTimeline() {
	suntimes := t.days.GetSuntimes(*t.currentDate)

	special := []timestampStyle{}
	cursorTime := t.TimeAtY(t.cursor.Y)
	cursorStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGray).Bold(true)
	if suntimes != nil && (cursorTime.IsAfter(suntimes.Set) || suntimes.Rise.IsAfter(cursorTime)) {
		cursorStyle = cursorStyle.Background(tcell.ColorBlack)
	}
	special = append(special, timestampStyle{cursorTime, cursorStyle})
	if t.currentDate.Is(time.Now()) {
		nowTime := *model.NewTimestampFromGotime(time.Now())
		nowStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorRed).Bold(true)
		special = append(special, timestampStyle{nowTime, nowStyle})
	}

	x := t.uiDimensions.TimelineOffset()
	y := 0
	w := t.uiDimensions.TimelineWidth()
	_, h := t.renderer.GetScreenDimensions()

	t.drawTimelineTmp(x, y, w, h, special, suntimes)
}

// Draw a timeline in the TUI at the provided coordinates in the provided
// dimensions.
// Optionally provide highlight times such as for the current timestamp as well
// as suntimes to be displayed on the timeline.
func (t *TUI) drawTimelineTmp(
	x, y, w, h int,
	highlightTimes []timestampStyle,
	suntimes *model.SunTimes) {

	timestampLength := 5
	timestampLPad := strings.Repeat(" ", w-timestampLength-1)
	timestampRPad := " "
	emptyTimestamp := strings.Repeat(" ", timestampLength)
	defaultStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGray)

	if t.viewParams.NRowsPerHour == 0 {
		panic("RES IS ZERO?!")
	}

	for virtRow := 0; virtRow <= h; virtRow++ {
		timestamp := t.TimeAtY(virtRow)

		if timestamp.Hour >= 24 {
			break
		}

		var timestampString string
		if timestamp.Minute == 0 {
			timestampString = timestamp.ToString()
		} else {
			timestampString = emptyTimestamp
		}
		timeText := timestampLPad + timestampString + timestampRPad

		var style tcell.Style
		if suntimes != nil && (!(timestamp.IsAfter(suntimes.Rise)) || (timestamp.IsAfter(suntimes.Set))) {
			style = defaultStyle.Background(tcell.ColorBlack)
		} else {
			style = defaultStyle
		}

		t.renderer.DrawText(x, virtRow+y, w, 1, style, timeText)
	}
	for _, timestampStyle := range highlightTimes {
		timestamp := timestampStyle.timestamp
		style := timestampStyle.style
		timeText := timestampLPad + timestamp.ToString() + timestampRPad
		t.renderer.DrawText(x, t.toY(timestamp)+y, w, 1, style, timeText)
	}
}

func (t *TUI) drawEvents() {
	day := t.days.GetDay(*t.currentDate)
	if day == nil {
		t.logWriter.Add("DEBUG", "current day nil on render; skipping")
		return
	}
	t.positions = t.ComputeRects(day, t.uiDimensions.EventsOffset(), 0, t.uiDimensions.EventsWidth()-2, t.uiDimensions.screenHeight)
	for _, e := range day.Events {
		style, err := t.categories.GetStyle(e.Cat)
		if err != nil {
			t.logWriter.Add("ERROR", err.Error())
		}
		// based on event state, draw a box or maybe a smaller one, or ...
		p := t.positions[e.ID]
		hovered := t.getEventForPos(t.cursor.X, t.cursor.Y)
		if hovered.Event != e.ID {
			t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H)
			t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, style, util.TruncateAt(e.Name, p.W-7))
			t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
			t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
		} else {
			selStyle := colors.DefaultEmphasize(style)
			switch hovered.HoverState {
			case ui.EventHoverStateResize:
				t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H-1)
				t.renderer.DrawBox(selStyle, p.X, p.Y+p.H-1, p.W, 1)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, style, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case ui.EventHoverStateMove:
				t.renderer.DrawBox(selStyle, p.X, p.Y, p.W, p.H)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, selStyle, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, selStyle, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case ui.EventHoverStateEdit:
				t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, selStyle, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
			default:
				panic("don't know this hover state!")
			}
		}
	}
}

// TODO: remove, this will be part of info returned to controller on query
func (t *TUI) TimeAtY(y int) model.Timestamp {
	minutes := y*(60/t.viewParams.NRowsPerHour) + t.viewParams.ScrollOffset*(60/t.viewParams.NRowsPerHour)

	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

func (t *TUI) toY(ts model.Timestamp) int {
	return ((ts.Hour*t.viewParams.NRowsPerHour - t.viewParams.ScrollOffset) + (ts.Minute / (60 / t.viewParams.NRowsPerHour)))
}

func (t *TUI) ComputeRects(day *model.Day, offsetX, offsetY, width, height int) map[model.EventID]util.Rect {
	active_stack := make([]model.Event, 0)
	positions := make(map[model.EventID]util.Rect)
	for _, e := range day.Events {
		// remove all stacked elements that have finished
		for i := len(active_stack) - 1; i >= 0; i-- {
			if e.Start.IsAfter(active_stack[i].End) || e.Start == active_stack[i].End {
				active_stack = active_stack[:i]
			} else {
				break
			}
		}
		active_stack = append(active_stack, e)
		// based on event state, draw a box or maybe a smaller one, or ...
		y := t.toY(e.Start) + offsetY
		x := offsetX
		h := t.toY(e.End) + offsetY - y
		w := width

		// scale the width by 3/4 for every extra item on the stack, so for one
		// item stacked underneath the current items width will be (3/4) ** 1 = 75%
		// of the original width, for four it would be (3/4) ** 4 = (3**4)/(4**4)
		// or 31.5 % of the width, etc.
		widthFactor := 0.75
		w = int(float64(w) * math.Pow(widthFactor, float64(len(active_stack)-1)))
		x += (width - w)

		positions[e.ID] = util.Rect{X: x, Y: y, W: w, H: h}
	}
	return positions
}

func (t *TUI) timeForDistance(dist int) model.TimeOffset {
	add := true
	if dist < 0 {
		dist *= (-1)
		add = false
	}
	minutes := dist * (60 / t.viewParams.NRowsPerHour)
	return model.TimeOffset{T: model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}, Add: add}
}
