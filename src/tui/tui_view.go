package tui

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/ja-he/dayplan/src/colors"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/util"

	"github.com/gdamore/tcell/v2"
)

type MainTUIPanel struct {
	renderer *TUIRenderer
	Model    *TUIModel
}

func (p *MainTUIPanel) Close() {
	p.renderer.Fini()
}

func (v *MainTUIPanel) NeedsSync() {
	v.renderer.NeedsSync()
}

type TUIRenderer struct {
	screen    tcell.Screen
	needsSync bool
}

func NewTUIRenderer() *TUIRenderer {
	r := &TUIRenderer{}
	r.init()

	return r
}

// Initialize the screen checking errors and return it, so long as no critical
// error occurred.
func (r *TUIRenderer) init() {
	var err error
	r.screen, err = tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	err = r.screen.Init()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	r.screen.SetStyle(defStyle)
	r.screen.EnableMouse()
	r.screen.EnablePaste()
	r.screen.Clear()
}

func (r *TUIRenderer) GetEventPollable() *tcell.Screen {
	return &r.screen
}

func (r *TUIRenderer) Fini() {
	r.screen.Fini()
}

func (r *TUIRenderer) NeedsSync() {
	r.needsSync = true
}

func (r *TUIRenderer) GetScreenDimensions() (int, int) {
	r.screen.SetStyle(tcell.StyleDefault)
	return r.screen.Size()
}

func (r *TUIRenderer) ShowCursor(x, y int) {
	r.screen.ShowCursor(x, y)
}

func (r *TUIRenderer) HideCursor() {
	r.screen.HideCursor()
}

func (r *TUIRenderer) Clear() {
	r.screen.Clear()
}

func (r *TUIRenderer) Show() {
	if r.needsSync {
		r.needsSync = false
		r.screen.Sync()
	} else {
		r.screen.Show()
	}
}

func NewTUIView(model *TUIModel, renderer *TUIRenderer) *MainTUIPanel {
	t := MainTUIPanel{}
	t.renderer = renderer

	t.Model = model
	w, h := t.renderer.GetScreenDimensions()
	weather, timeline, tools := 20, 10, 20
	t.Model.UIDim.Initialize(weather, timeline, tools, w, h)

	return &t
}

const editorWidth = 80
const editorHeight = 20

const helpWidth = 80
const helpHeight = 30

func (t *MainTUIPanel) getScreenCenter() (int, int) {
	w, h := t.renderer.GetScreenDimensions()
	x := w / 2
	y := h / 2
	return x, y
}

func (t *MainTUIPanel) DrawTools() {
	i := 0

	boxes := t.Model.CalculateCategoryBoxes()
	for _, styling := range t.Model.CategoryStyling.GetAll() {
		box := boxes[styling.Cat]
		textHeightOffset := box.H / 2
		textLen := box.W - 2

		t.renderer.DrawBox(styling.Style, box.X, box.Y, box.W, box.H)
		t.renderer.DrawText(box.X+1, box.Y+textHeightOffset, textLen, 0, styling.Style, util.TruncateAt(styling.Cat.Name, textLen))
		if t.Model.CurrentCategory.Name == styling.Cat.Name {
			t.renderer.DrawBox(colors.DefaultEmphasize(styling.Style), box.X+box.W-1, box.Y, 1, box.H)
		}

		i++
	}
}

func (t *MainTUIPanel) DrawEditor() {
	editor := &t.Model.EventEditor
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
func (t *MainTUIPanel) DrawHelp() {
	if t.Model.showHelp {

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
func (t *MainTUIPanel) DrawSummary() {
	style := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	if t.Model.showSummary {
		y, w, h := 2, t.Model.UIDim.screenWidth, t.Model.UIDim.screenHeight
		t.renderer.DrawBox(style, 0, 0, w, h)
		dateString := ""
		switch t.Model.activeView {
		case ViewDay:
			dateString = t.Model.CurrentDate.ToString()
		case ViewWeek:
			start, end := t.Model.CurrentDate.Week()
			dateString = fmt.Sprintf("week %s..%s", start.ToString(), end.ToString())
		case ViewMonth:
			dateString = fmt.Sprintf("%s %d", t.Model.CurrentDate.ToGotime().Month().String(), t.Model.CurrentDate.Year)
		}
		title := fmt.Sprintf("SUMMARY (%s)", dateString)
		t.renderer.DrawBox(style.Background(tcell.ColorLightGrey), 0, 0, w, 1)
		t.renderer.DrawText(w/2-len(title)/2, 0, len(title), 1, style.Background(tcell.ColorLightGrey).Bold(true), title)

		summary := make(map[model.Category]int)
		switch t.Model.activeView {
		case ViewDay:
			day := t.Model.GetCurrentDay()
			if day == nil {
				return
			}
			summary = day.SumUpByCategory()
		case ViewWeek:
			start, end := t.Model.CurrentDate.Week()
			for current := start; current != end.Next(); current = current.Next() {
				day := t.Model.GetDay(current)
				if day == nil {
					return
				}
				tmpSummary := day.SumUpByCategory()
				for k, v := range tmpSummary {
					summary[k] += v
				}
			}
		case ViewMonth:
			start, end := t.Model.CurrentDate.MonthBounds()
			for current := start; current != end.Next(); current = current.Next() {
				day := t.Model.GetDay(current)
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
			style, _ := t.Model.CategoryStyling.GetStyle(category)
			catLen := 20
			durationLen := 20
			barWidth := int(float64(duration) / float64(maxDuration) * float64(t.Model.UIDim.screenWidth-catLen-durationLen))
			t.renderer.DrawBox(style, catLen+durationLen, y, barWidth, 1)
			t.renderer.DrawText(0, y, catLen, 0, tcell.StyleDefault, util.TruncateAt(category.Name, catLen))
			t.renderer.DrawText(catLen, y, durationLen, 0, style, "("+util.DurationToString(duration)+")")
			y++
		}
	}
}

func (t *MainTUIPanel) DrawLog() {
	style := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	if t.Model.showLog {
		x, y, w, h := 0, 2, t.Model.UIDim.screenWidth, t.Model.UIDim.screenHeight
		t.renderer.DrawBox(style, 0, 0, w, h)
		title := "LOG"
		t.renderer.DrawBox(style.Background(tcell.ColorLightGrey), 0, 0, w, 1)
		t.renderer.DrawText(w/2-len(title)/2, 0, len(title), 1, style.Background(tcell.ColorLightGrey).Bold(true), title)
		for i := len(t.Model.Log.Get()) - 1; i >= 0; i-- {
			entry := &t.Model.Log.Get()[i]

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

func (t *MainTUIPanel) Draw(x, y, w, h int) {

	t.renderer.Clear()

	// TODO: define all styles here (prep to probably move out further)
	headerBG := tcell.StyleDefault.Background(colors.ColorFromHexString("#f0f0f0")).Foreground(tcell.ColorBlack)
	headerBGEmph := colors.DefaultEmphasize(headerBG)
	dayBG := tcell.StyleDefault
	dayBGEmph := headerBGEmph
	loadingStyle := dayBG.Foreground(tcell.ColorLightSeaGreen)

	switch t.Model.activeView {
	case ViewDay:
		t.DrawWeather()
		t.DrawTimeline()
		t.DrawEvents()
		t.DrawTools()
		t.DrawEditor()
	case ViewWeek:
		start, end := t.Model.CurrentDate.Week()
		nDays := start.DaysUntil(end) + 1
		if nDays > t.Model.UIDim.screenWidth {
			t.renderer.DrawText(0, 0, t.Model.UIDim.screenWidth, t.Model.UIDim.screenHeight,
				tcell.StyleDefault.Foreground(tcell.ColorRebeccaPurple),
				"refusing to render week on screen with fewer columns than days")
			return
		}

		{
			firstDayXOffset := 10
			x := firstDayXOffset
			dayWidth := (t.Model.UIDim.screenWidth - firstDayXOffset) / nDays

			t.drawTimeline(0, 0, firstDayXOffset, t.Model.UIDim.screenHeight-t.Model.UIDim.statusHeight, make([]timestampStyle, 0), nil)

			for drawDate := start; drawDate != end.Next(); drawDate = drawDate.Next() {
				if drawDate == t.Model.CurrentDate {
					t.renderer.DrawBox(dayBGEmph, x, 0, dayWidth, t.Model.UIDim.screenHeight)
				} else {
					t.renderer.DrawBox(dayBG, x, 0, dayWidth, t.Model.UIDim.screenHeight)
				}
				day := t.Model.GetDay(drawDate)
				if day != nil {
					positions := t.Model.ComputeRects(day, x, 0, dayWidth, t.Model.UIDim.screenHeight-t.Model.UIDim.statusHeight)
					for _, e := range day.Events {
						p := positions[e.ID]
						style, err := t.Model.CategoryStyling.GetStyle(e.Cat)
						if err != nil {
							panic(err)
						}
						if drawDate != t.Model.CurrentDate {
							style = colors.DefaultDim(style)
						}
						t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H)
						t.renderer.DrawText(p.X, p.Y, p.W, 0, style, util.TruncateAt(e.Name, p.W))
					}
				} else {
					loadingText := "⋮"
					t.renderer.DrawText(x, t.Model.UIDim.screenHeight/2-len([]rune(loadingText)), 1, len([]rune(loadingText)),
						loadingStyle,
						loadingText)
				}
				x += dayWidth
			}
		}

	case ViewMonth:
		start, end := t.Model.CurrentDate.MonthBounds()
		nDays := start.DaysUntil(end) + 1
		if nDays > t.Model.UIDim.screenWidth {
			t.renderer.DrawText(0, 0, t.Model.UIDim.screenWidth, t.Model.UIDim.screenHeight,
				tcell.StyleDefault.Foreground(tcell.ColorRebeccaPurple),
				"refusing to render month on screen with fewer columns than days")
			return
		}

		{
			firstDayXOffset := 10
			x := firstDayXOffset
			dayWidth := (t.Model.UIDim.screenWidth - firstDayXOffset) / nDays

			t.drawTimeline(0, 0, firstDayXOffset, t.Model.UIDim.screenHeight-t.Model.UIDim.statusHeight, make([]timestampStyle, 0), nil)

			for drawDate := start; drawDate != end.Next(); drawDate = drawDate.Next() {
				if drawDate == t.Model.CurrentDate {
					t.renderer.DrawBox(dayBGEmph, x, 0, dayWidth, t.Model.UIDim.screenHeight)
				} else {
					t.renderer.DrawBox(dayBG, x, 0, dayWidth, t.Model.UIDim.screenHeight)
				}
				day := t.Model.GetDay(drawDate)
				if day != nil {
					positions := t.Model.ComputeRects(day, x, 0, dayWidth, t.Model.UIDim.screenHeight-t.Model.UIDim.statusHeight)
					for _, e := range day.Events {
						p := positions[e.ID]
						style, err := t.Model.CategoryStyling.GetStyle(e.Cat)
						if err != nil {
							panic(err)
						}
						if drawDate != t.Model.CurrentDate {
							style = colors.DefaultDim(style)
						}
						t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H)
					}
				} else {
					loadingText := "⋮"
					t.renderer.DrawText(x, t.Model.UIDim.screenHeight/2-len([]rune(loadingText)), 1, len([]rune(loadingText)),
						loadingStyle,
						loadingText)
				}
				x += dayWidth
			}
		}

	default:
		t.Model.Log.Add("ERROR", fmt.Sprintf("unknown active view %d aka '%s'",
			t.Model.activeView, toString(t.Model.activeView)))
	}
	t.DrawStatus()
	t.DrawLog()
	t.DrawSummary()
	t.DrawHelp()

	t.renderer.Show()
}

func (t *TUIRenderer) DrawText(x, y, w, h int, style tcell.Style, text string) {
	row := y
	col := x
	for _, r := range text {
		t.screen.SetContent(col, row, r, nil, style)
		col++
		if col >= x+w {
			row++
			col = x
		}
		if row > y+h {
			break
		}
	}
}

func (t *TUIRenderer) DrawBox(style tcell.Style, x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			t.screen.SetContent(col, row, ' ', nil, style)
		}
	}
}

func (t *MainTUIPanel) DrawStatus() {
	var firstDay, lastDay model.Date
	switch t.Model.activeView {
	case ViewDay:
		firstDay, lastDay = t.Model.CurrentDate, t.Model.CurrentDate
	case ViewWeek:
		firstDay, lastDay = t.Model.CurrentDate.Week()
	case ViewMonth:
		firstDay, lastDay = t.Model.CurrentDate.MonthBounds()
	}

	firstDayXOffset := 10
	nDaysInPeriod := firstDay.DaysUntil(lastDay) + 1
	nDaysTilCurrent := firstDay.DaysUntil(t.Model.CurrentDate)
	dateWidth := 10 // 2020-02-12 is 10 wide
	dayWidth := (t.Model.UIDim.screenWidth - firstDayXOffset) / nDaysInPeriod
	statusYOffset := t.Model.UIDim.StatusOffset()

	bgStyle := tcell.StyleDefault.Background(colors.ColorFromHexString("#f0f0f0")).Foreground(tcell.ColorBlack)
	bgStyleEmph := colors.DefaultEmphasize(bgStyle)
	dateStyle := bgStyleEmph
	weekdayStyle := colors.LightenFG(dateStyle, 60)

	// header background
	t.renderer.DrawBox(bgStyle, 0, statusYOffset, firstDayXOffset+nDaysInPeriod*dayWidth, t.Model.UIDim.statusHeight)
	// header bar (filled for days until current)
	t.renderer.DrawBox(bgStyleEmph, 0, statusYOffset, firstDayXOffset+(nDaysTilCurrent+1)*dayWidth, t.Model.UIDim.statusHeight)
	// date box background
	t.renderer.DrawBox(bgStyleEmph, 0, statusYOffset, dateWidth, t.Model.UIDim.statusHeight)
	// date string
	t.renderer.DrawText(0, statusYOffset, dateWidth, 0, dateStyle, t.Model.CurrentDate.ToString())
	// weekday string
	t.renderer.DrawText(0, statusYOffset+1, dateWidth, 0, weekdayStyle, util.TruncateAt(t.Model.CurrentDate.ToWeekday().String(), dateWidth))
}

func (t *MainTUIPanel) DrawWeather() {
	for timestamp := *model.NewTimestamp("00:00"); timestamp.Legal(); timestamp.Hour++ {
		y := t.Model.toY(timestamp)

		index := model.DayAndTime{
			Date:      t.Model.CurrentDate,
			Timestamp: timestamp,
		}

		weather, ok := t.Model.Weather.Data[index]
		if ok {
			weatherStyle := tcell.StyleDefault.Foreground(tcell.ColorLightBlue)
			switch {
			case weather.PrecipitationProbability > .25:
				weatherStyle = weatherStyle.Background(tcell.NewHexColor(0xccebff)).Foreground(tcell.ColorBlack)
			case weather.Clouds < 25:
				weatherStyle = weatherStyle.Background(tcell.NewHexColor(0xfff0cc)).Foreground(tcell.ColorBlack)
			}

			t.renderer.DrawBox(weatherStyle, t.Model.UIDim.WeatherOffset(), y, t.Model.UIDim.WeatherWidth(), t.Model.NRowsPerHour)

			t.renderer.DrawText(t.Model.UIDim.WeatherOffset(), y, t.Model.UIDim.WeatherWidth(), 0, weatherStyle, weather.Info)
			t.renderer.DrawText(t.Model.UIDim.WeatherOffset(), y+1, t.Model.UIDim.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%2.0f°C", weather.TempC))
			t.renderer.DrawText(t.Model.UIDim.WeatherOffset(), y+2, t.Model.UIDim.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%d%% clouds", weather.Clouds))
			t.renderer.DrawText(t.Model.UIDim.WeatherOffset(), y+3, t.Model.UIDim.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%d%% humidity", weather.Humidity))
			t.renderer.DrawText(t.Model.UIDim.WeatherOffset(), y+4, t.Model.UIDim.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%2.0f%% chance of rain", 100.0*weather.PrecipitationProbability))
		}
	}
}

type timestampStyle struct {
	timestamp model.Timestamp
	style     tcell.Style
}

func (t *MainTUIPanel) DrawTimeline() {
	suntimes := t.Model.GetCurrentSuntimes()

	special := []timestampStyle{}
	cursorTime := t.Model.TimeAtY(t.Model.cursorY)
	cursorStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGray).Bold(true)
	if suntimes != nil && (cursorTime.IsAfter(suntimes.Set) || suntimes.Rise.IsAfter(cursorTime)) {
		cursorStyle = cursorStyle.Background(tcell.ColorBlack)
	}
	special = append(special, timestampStyle{cursorTime, cursorStyle})
	if t.Model.CurrentDate.Is(time.Now()) {
		nowTime := *model.NewTimestampFromGotime(time.Now())
		nowStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorRed).Bold(true)
		special = append(special, timestampStyle{nowTime, nowStyle})
	}

	x := t.Model.UIDim.TimelineOffset()
	y := 0
	w := t.Model.UIDim.TimelineWidth()
	_, h := t.renderer.GetScreenDimensions()

	t.drawTimeline(x, y, w, h, special, suntimes)
}

// Draw a timeline in the TUI at the provided coordinates in the provided
// dimensions.
// Optionally provide highlight times such as for the current timestamp as well
// as suntimes to be displayed on the timeline.
func (t *MainTUIPanel) drawTimeline(
	x, y, w, h int,
	highlightTimes []timestampStyle,
	suntimes *model.SunTimes) {

	timestampLength := 5
	timestampLPad := strings.Repeat(" ", w-timestampLength-1)
	timestampRPad := " "
	emptyTimestamp := strings.Repeat(" ", timestampLength)
	defaultStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGray)

	if t.Model.NRowsPerHour == 0 {
		panic("RES IS ZERO?!")
	}

	for virtRow := 0; virtRow <= h; virtRow++ {
		timestamp := t.Model.TimeAtY(virtRow)

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
		t.renderer.DrawText(x, t.Model.toY(timestamp)+y, w, 1, style, timeText)
	}
}

func (t *MainTUIPanel) DrawEvents() {
	day := t.Model.GetCurrentDay()
	if day == nil {
		t.Model.Log.Add("DEBUG", "current day nil on render; skipping")
		return
	}
	t.Model.Positions = t.Model.ComputeRects(day, t.Model.UIDim.EventsOffset(), 0, t.Model.UIDim.EventsWidth()-2, t.Model.UIDim.screenHeight)
	for _, e := range day.Events {
		style, err := t.Model.CategoryStyling.GetStyle(e.Cat)
		if err != nil {
			t.Model.Log.Add("ERROR", err.Error())
		}
		// based on event state, draw a box or maybe a smaller one, or ...
		p := t.Model.Positions[e.ID]
		if t.Model.Hovered.EventID != e.ID {
			t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H)
			t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, style, util.TruncateAt(e.Name, p.W-7))
			t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
			t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
		} else {
			selStyle := colors.DefaultEmphasize(style)
			switch t.Model.Hovered.HoverState {
			case HoverStateResize:
				t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H-1)
				t.renderer.DrawBox(selStyle, p.X, p.Y+p.H-1, p.W, 1)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, style, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case HoverStateMove:
				t.renderer.DrawBox(selStyle, p.X, p.Y, p.W, p.H)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, selStyle, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, selStyle, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case HoverStateEdit:
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
