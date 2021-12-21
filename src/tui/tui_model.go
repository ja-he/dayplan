package tui

import (
	"fmt"
	"math"
	"sync"

	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/program"
	"github.com/ja-he/dayplan/src/util"
	"github.com/ja-he/dayplan/src/weather"
)

type hoveredEventInfo struct {
	EventID    model.EventID
	HoverState HoverState
}

type UIPane int

const (
	UIWeather UIPane = iota
	UITimeline
	UIEvents
	UITools
	UIStatus
)

type ActiveView int

const (
	_ ActiveView = iota
	ViewDay
	ViewWeek
	ViewMonth
)

func PrevView(current ActiveView) ActiveView {
	switch current {
	case ViewDay:
		return ViewWeek
	case ViewWeek:
		return ViewMonth
	case ViewMonth:
		return ViewMonth
	default:
		panic("unknown view!")
	}
}

func NextView(current ActiveView) ActiveView {
	switch current {
	case ViewDay:
		return ViewDay
	case ViewWeek:
		return ViewDay
	case ViewMonth:
		return ViewWeek
	default:
		panic("unknown view!")
	}
}

func toString(av ActiveView) string {
	switch av {
	case ViewDay:
		return "ViewDay"
	case ViewWeek:
		return "ViewWeek"
	case ViewMonth:
		return "ViewMonth"
	default:
		return "unknown"
	}
}

type UIDims struct {
	weatherOffset, timelineOffset, eventsOffset, toolsOffset int
	statusHeight                                             int
	screenWidth, screenHeight                                int
}

func (d *UIDims) WhichUIPane(x, y int) UIPane {
	if x < 0 || y < 0 {
		panic("negative x or y")
	}
	if x > d.screenWidth || y > d.screenHeight {
		panic(fmt.Sprintf("x or y too large (x,y = %d,%d | screen = %d,%d)", x, y, d.screenWidth, d.screenHeight))
	}

	mainPaneHeight := d.screenHeight - d.StatusHeight()

	statusPane := util.Rect{X: 0, Y: d.StatusOffset(), W: d.screenWidth, H: d.StatusHeight()}
	weatherPane := util.Rect{X: d.WeatherOffset(), Y: 0, W: d.WeatherWidth(), H: mainPaneHeight}
	timelinePane := util.Rect{X: d.TimelineOffset(), Y: 0, W: d.TimelineWidth(), H: mainPaneHeight}
	eventsPane := util.Rect{X: d.EventsOffset(), Y: 0, W: d.EventsWidth(), H: mainPaneHeight}
	toolsPane := util.Rect{X: d.ToolsOffset(), Y: 0, W: d.ToolsWidth(), H: mainPaneHeight}

	if statusPane.Contains(x, y) {
		return UIStatus
	}
	if weatherPane.Contains(x, y) {
		return UIWeather
	}
	if timelinePane.Contains(x, y) {
		return UITimeline
	}
	if eventsPane.Contains(x, y) {
		return UIEvents
	}
	if toolsPane.Contains(x, y) {
		return UITools
	}

	panic(fmt.Sprintf("Unknown UI pos (%d,%d)", x, y))
}

func (d *UIDims) ScreenSize() (int, int) {
	return d.screenWidth, d.screenHeight
}

func (d *UIDims) ScreenResize(width, height int) {
	if height <= d.statusHeight {
		panic(fmt.Sprintf("screensize of %d too little with statusline height of %d", height, d.statusHeight))
	}

	toolsWidth := d.ToolsWidth()
	d.toolsOffset = width - toolsWidth
	if d.toolsOffset >= width {
		panic("offset > width")
	}

	d.screenWidth = width
	d.screenHeight = height
}

func (d *UIDims) Initialize(weatherWidth, timelineWidth, toolsWidth int,
	screenWidth, screenHeight int) {
	d.weatherOffset = 0
	d.timelineOffset = d.weatherOffset + weatherWidth
	d.eventsOffset = d.timelineOffset + timelineWidth
	d.toolsOffset = screenWidth - toolsWidth
	d.statusHeight = 1
	d.screenWidth = screenWidth
	d.screenHeight = screenHeight
}

func (ui *UIDims) WeatherOffset() int  { return ui.weatherOffset }
func (ui *UIDims) WeatherWidth() int   { return ui.timelineOffset - ui.weatherOffset }
func (ui *UIDims) TimelineOffset() int { return ui.timelineOffset }
func (ui *UIDims) TimelineWidth() int  { return ui.eventsOffset - ui.timelineOffset }
func (ui *UIDims) EventsOffset() int   { return ui.eventsOffset }
func (ui *UIDims) EventsWidth() int    { return (ui.toolsOffset - ui.eventsOffset) }
func (ui *UIDims) ToolsOffset() int    { return ui.toolsOffset }
func (ui *UIDims) ToolsWidth() int     { return ui.screenWidth - ui.ToolsOffset() }
func (ui *UIDims) StatusHeight() int   { return ui.statusHeight }
func (ui *UIDims) StatusOffset() int   { return ui.screenHeight - ui.statusHeight }

type DayWithInfo struct {
	Day      *model.Day
	SunTimes *model.SunTimes
}

type TUIModel struct {
	UIDim            UIDims
	CategoryStyling  category_style.CategoryStyling
	Positions        map[model.EventID]util.Rect
	Hovered          hoveredEventInfo
	cursorX, cursorY int
	daysMutex        sync.RWMutex
	days             map[model.Date]DayWithInfo
	CurrentDate      model.Date
	Log              potatolog.Log
	showLog          bool
	Resolution       int
	ScrollOffset     int
	EventEditor      EventEditor
	showSummary      bool
	Weather          weather.Handler
	CurrentCategory  model.Category
	ProgramData      program.Data
	activeView       ActiveView
}

func (t *TUIModel) ScrollUp(by int) {
	if t.ScrollOffset-by >= 0 {
		t.ScrollOffset -= by
	} else {
		t.ScrollOffset = 0
	}
}

func (t *TUIModel) ScrollDown(by int) {
	eventviewBottomRow := t.UIDim.screenHeight - t.UIDim.statusHeight
	if t.ScrollOffset+by+eventviewBottomRow <= (24 * t.Resolution) {
		t.ScrollOffset += by
	} else {
		t.ScrollOffset = (24 * t.Resolution) - eventviewBottomRow
	}
}

func (t *TUIModel) ScrollTop() {
	t.ScrollOffset = 0
}

func (t *TUIModel) ScrollBottom() {
	eventviewBottomRow := t.UIDim.screenHeight - t.UIDim.statusHeight
	t.ScrollOffset = 24*t.Resolution - eventviewBottomRow
}

func NewTUIModel(cs category_style.CategoryStyling) *TUIModel {
	var t TUIModel

	t.days = make(map[model.Date]DayWithInfo)

	t.CategoryStyling = cs
	t.Positions = make(map[model.EventID]util.Rect)

	t.Resolution = 6
	t.ScrollOffset = 8 * t.Resolution

	t.activeView = ViewDay

	return &t
}

func (t *TUIModel) TimeForDistance(dist int) model.TimeOffset {
	add := true
	if dist < 0 {
		dist *= (-1)
		add = false
	}
	minutes := dist * (60 / t.Resolution)
	return model.TimeOffset{T: model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}, Add: add}
}

// TODO: rename HasDay
func (t *TUIModel) HasModel(date model.Date) bool {
	t.daysMutex.RLock()
	defer t.daysMutex.RUnlock()
	_, ok := t.days[date]
	return ok
}

func (t *TUIModel) GetCurrentDay() *model.Day {
	return t.GetDay(t.CurrentDate)
}

// Get the suntimes of the current date of the model.
func (t *TUIModel) GetCurrentSuntimes() *model.SunTimes {
	return t.GetSuntimes(t.CurrentDate)
}

// Get the suntimes of the provided date of the model.
func (t *TUIModel) GetSuntimes(date model.Date) *model.SunTimes {
	t.daysMutex.RLock()
	defer t.daysMutex.RUnlock()
	return t.days[date].SunTimes
}

// Get the day of the provided date of the model.
func (t *TUIModel) GetDay(date model.Date) *model.Day {
	t.daysMutex.RLock()
	defer t.daysMutex.RUnlock()
	return t.days[date].Day
}

// TODO: rename AddDay
func (t *TUIModel) AddModel(date model.Date, day *model.Day, suntimes *model.SunTimes) {
	if day == nil {
		panic("will not add a nil model")
	}
	t.Log.Add("DEBUG", "adding non-nil model for day "+date.ToString())

	t.daysMutex.Lock()
	defer t.daysMutex.Unlock()
	t.days[date] = DayWithInfo{day, suntimes}
}

func (t *TUIModel) TimeAtY(y int) model.Timestamp {
	minutes := y*(60/t.Resolution) + t.ScrollOffset*(60/t.Resolution)

	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

func (t *TUIModel) ComputeRects(day *model.Day, offsetX, width int) map[model.EventID]util.Rect {
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
		y := t.toY(e.Start)
		x := offsetX
		h := t.toY(e.End) - y
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

func NoHoveredEvent() hoveredEventInfo {
	return hoveredEventInfo{0, HoverStateNone}
}

// TODO: move to controller?
func (t *TUIModel) GetEventForPos(x, y int) hoveredEventInfo {
	if x >= t.UIDim.EventsOffset() &&
		x < (t.UIDim.EventsOffset()+t.UIDim.EventsWidth()) {
		for i := len(t.GetCurrentDay().Events) - 1; i >= 0; i-- {
			eventPos := t.Positions[t.GetCurrentDay().Events[i].ID]
			if eventPos.Contains(x, y) {
				var hover HoverState
				switch {
				case y == (eventPos.Y+eventPos.H-1) && x > eventPos.X+eventPos.W-5:
					hover = HoverStateResize
				case y == (eventPos.Y):
					hover = HoverStateEdit
				default:
					hover = HoverStateMove
				}
				return hoveredEventInfo{t.GetCurrentDay().Events[i].ID, hover}
			}
		}
	}
	return NoHoveredEvent()
}

// TODO: wtf is this supposed to be good for?!
func (t *TUIModel) CalculateCategoryBoxes() map[model.Category]util.Rect {
	day := make(map[model.Category]util.Rect)

	offsetX := 1
	offsetY := 1
	gap := 0

	for i, c := range t.CategoryStyling.GetAll() {
		day[c.Cat] = util.Rect{
			X: t.UIDim.ToolsOffset() + offsetX,
			Y: offsetY + (i) + (i * gap),
			W: t.UIDim.ToolsWidth() - (2 * offsetX),
			H: 1,
		}
	}

	return day
}

func (t *TUIModel) GetCategoryForPos(x, y int) *model.Category {
	boxes := t.CalculateCategoryBoxes()

	for cat, box := range boxes {
		if box.Contains(x, y) {
			return &cat
		}
	}

	return nil
}

func (t *TUIModel) ClearHover() {
	t.Hovered = NoHoveredEvent()
}

func (t *TUIModel) toY(ts model.Timestamp) int {
	return ((ts.Hour*t.Resolution - t.ScrollOffset) + (ts.Minute / (60 / t.Resolution)))
}
