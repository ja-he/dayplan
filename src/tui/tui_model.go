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

type Status struct {
	status map[string]string
	mutex  sync.Mutex
}

func (s *Status) Set(key, val string) {
	s.mutex.Lock()
	s.status[key] = val
	s.mutex.Unlock()
}

func (s *Status) Get() map[string]string {
	return s.status
}

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
	Days             map[model.Date]DayWithInfo
	CurrentDate      model.Date
	Status           Status
	Log              potatolog.Log
	showLog          bool
	Resolution       int
	ScrollOffset     int
	EventEditor      EventEditor
	Weather          weather.Handler
	CurrentCategory  model.Category
	ProgramData      program.Data
}

func (t *TUIModel) ScrollUp(by int) {
	if t.ScrollOffset-by >= 0 {
		t.ScrollOffset -= by
	}
}

func (t *TUIModel) ScrollDown(by int) {
	eventviewBottomRow := t.UIDim.screenHeight - t.UIDim.statusHeight
	if t.ScrollOffset+by+eventviewBottomRow <= (24 * t.Resolution) {
		t.ScrollOffset += by
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
	t.Status.status = make(map[string]string)

	t.Days = make(map[model.Date]DayWithInfo)

	t.CategoryStyling = cs
	t.Positions = make(map[model.EventID]util.Rect)

	t.Resolution = 6
	t.ScrollOffset = 8 * t.Resolution

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

func (t *TUIModel) HasModel(date model.Date) bool {
	_, ok := t.Days[date]
	return ok
}

func (t *TUIModel) GetCurrentDay() *model.Day {
	// this _should_ always be available
	return t.Days[t.CurrentDate].Day
}

func (t *TUIModel) AddModel(date model.Date, day *model.Day, suntimes *model.SunTimes) {
	if day == nil {
		panic("will not add a nil model")
	}
	t.Log.Add("DEBUG", "adding non-nil model for day "+date.ToString())
	t.Days[date] = DayWithInfo{day, suntimes}
}

func (t *TUIModel) TimeAtY(y int) model.Timestamp {
	minutes := y*(60/t.Resolution) + t.ScrollOffset*(60/t.Resolution)

	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

func (t *TUIModel) ComputeRects() {
	defaultX := t.UIDim.EventsOffset()
	defaultW := t.UIDim.EventsWidth() - 2 // -2 so we have some space to the right to insert events

	active_stack := make([]model.Event, 0)
	for _, e := range t.GetCurrentDay().Events {
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
		x := defaultX
		h := t.toY(e.End) - y
		w := defaultW

		// scale the width by 3/4 for every extra item on the stack, so for one
		// item stacked underneath the current items width will be (3/4) ** 1 = 75%
		// of the original width, for four it would be (3/4) ** 4 = (3**4)/(4**4)
		// or 31.5 % of the width, etc.
		widthFactor := 0.75
		w = int(float64(w) * math.Pow(widthFactor, float64(len(active_stack)-1)))
		x += (defaultW - w)

		t.Positions[e.ID] = util.Rect{X: x, Y: y, W: w, H: h}
	}
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

const categoryBoxHeight int = 3

func (t *TUIModel) CalculateCategoryBoxes() map[model.Category]util.Rect {
	day := make(map[model.Category]util.Rect)

	offsetX := 1
	offsetY := 1
	gap := 0

	for i, c := range t.CategoryStyling.GetAll() {
		day[c.Cat] = util.Rect{
			X: t.UIDim.ToolsOffset() + offsetX,
			Y: offsetY + (i * categoryBoxHeight) + (i * gap),
			W: t.UIDim.ToolsWidth() - (2 * offsetX),
			H: categoryBoxHeight,
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
