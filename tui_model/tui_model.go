package tui_model

import (
	"fmt"
	"math"

	"dayplan/hover_state"
	"dayplan/model"
	"dayplan/timestamp"
	"dayplan/util"

	"github.com/gdamore/tcell/v2"
)

type hoveredEventInfo struct {
	EventID    model.EventID
	HoverState hover_state.HoverState
}

type UIPane int

const (
	Weather UIPane = iota
	Timeline
	Events
	Tools
	Status
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
		return Status
	}
	if weatherPane.Contains(x, y) {
		return Weather
	}
	if timelinePane.Contains(x, y) {
		return Timeline
	}
	if eventsPane.Contains(x, y) {
		return Events
	}
	if toolsPane.Contains(x, y) {
		return Tools
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
	d.statusHeight = 4
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

type TUIModel struct {
	UIDim           UIDims
	CategoryStyling map[model.Category]tcell.Style
	Positions       map[model.EventID]util.Rect
	Hovered         hoveredEventInfo
	Model           *model.Model
	Status          string
	Resolution      int
	ScrollOffset    int
}

func (t *TUIModel) ScrollUp() {
	if t.ScrollOffset > 0 {
		t.ScrollOffset -= t.Resolution
	}
}

func (t *TUIModel) ScrollDown() {
	if t.ScrollOffset < (23 * t.Resolution) {
		t.ScrollOffset += t.Resolution
	}
}

func NewTUIModel() *TUIModel {
	var t TUIModel

	t.CategoryStyling = make(map[model.Category]tcell.Style)
	t.Positions = make(map[model.EventID]util.Rect)
	t.CategoryStyling[model.Category{Name: "default"}] = tcell.StyleDefault.Background(tcell.NewHexColor(0xff00ff)).Foreground(tcell.NewHexColor(0x00ff00))
	t.CategoryStyling[model.Category{Name: "work"}] = tcell.StyleDefault.Background(tcell.NewHexColor(0xccebff)).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "leisure"}] = tcell.StyleDefault.Background(tcell.Color76).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "misc"}] = tcell.StyleDefault.Background(tcell.Color250).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "programming"}] = tcell.StyleDefault.Background(tcell.Color226).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "cooking"}] = tcell.StyleDefault.Background(tcell.Color212).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "fitness"}] = tcell.StyleDefault.Background(tcell.Color208).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "eating"}] = tcell.StyleDefault.Background(tcell.Color224).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "hygiene"}] = tcell.StyleDefault.Background(tcell.Color80).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "cleaning"}] = tcell.StyleDefault.Background(tcell.Color215).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "laundry"}] = tcell.StyleDefault.Background(tcell.Color111).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "family"}] = tcell.StyleDefault.Background(tcell.Color122).Foreground(tcell.ColorReset)
	t.Status = "initial status msg"

	t.Resolution = 6
	t.ScrollOffset = 8 * t.Resolution

	return &t
}

func (t *TUIModel) TimeForDistance(dist int) timestamp.TimeOffset {
	add := true
	if dist < 0 {
		dist *= (-1)
		add = false
	}
	minutes := dist * (60 / t.Resolution)
	return timestamp.TimeOffset{T: timestamp.Timestamp{Hour: minutes / 60, Minute: minutes % 60}, Add: add}
}

func (t *TUIModel) SetModel(m *model.Model) {
	t.Model = m
}

func (t *TUIModel) TimeAtY(y int) timestamp.Timestamp {
	minutes := y*(60/t.Resolution) + t.ScrollOffset*(60/t.Resolution)

	ts := timestamp.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

func (t *TUIModel) ComputeRects() {
	defaultX := t.UIDim.EventsOffset()
	defaultW := t.UIDim.EventsWidth()

	active_stack := make([]model.Event, 0)
	for _, e := range t.Model.Events {
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
	return hoveredEventInfo{0, hover_state.None}
}

// TODO: move to controller?
func (t *TUIModel) GetEventForPos(x, y int) hoveredEventInfo {
	if x >= t.UIDim.EventsOffset() &&
		x < (t.UIDim.EventsOffset()+t.UIDim.EventsWidth()) {
		for i := len(t.Model.Events) - 1; i >= 0; i-- {
			if t.Positions[t.Model.Events[i].ID].Contains(x, y) {
				if y == (t.Positions[t.Model.Events[i].ID].Y + t.Positions[t.Model.Events[i].ID].H - 1) {
					return hoveredEventInfo{t.Model.Events[i].ID, hover_state.Resize}
				} else {
					return hoveredEventInfo{t.Model.Events[i].ID, hover_state.Move}
				}
			}
		}
	}
	return NoHoveredEvent()
}

func (t *TUIModel) ClearHover() {
	t.Hovered = NoHoveredEvent()
}

func (t *TUIModel) toY(ts timestamp.Timestamp) int {
	return ((ts.Hour*t.Resolution - t.ScrollOffset) + (ts.Minute / (60 / t.Resolution)))
}
