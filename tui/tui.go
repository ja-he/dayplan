package tui

import (
	"fmt"
	"log"
	"time"

	"dayplan/model"
	"dayplan/timestamp"
	"dayplan/util"

	"github.com/gdamore/tcell/v2"
)

// Initialize the screen checking errors and return it, so long as no critical
// error occurred.
func initScreen() tcell.Screen {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	return s
}

type eventHoverState struct {
	Event  *model.Event
	Resize bool
}

type editState int

const (
	None editState = iota
	Moving
	Resizing
)

type TUI struct {
	Screen                          tcell.Screen
	CursorX, CursorY                int
	eventviewOffset, eventviewWidth int
	CategoryStyling                 map[model.Category]tcell.Style
	positions                       map[model.Event]util.Rect
	Hovered                         eventHoverState
	EditState                       editState
	model                           *model.Model
	EditedEvent                     *model.Event
	status                          string
	Resolution                      int
	ScrollOffset                    int
}

func NewTUI() *TUI {
	var t TUI

	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)

	t.Screen = initScreen()
	t.Screen.SetStyle(defStyle)
	t.Screen.EnableMouse()
	t.Screen.EnablePaste()
	t.Screen.Clear()
	t.CategoryStyling = make(map[model.Category]tcell.Style)
	t.positions = make(map[model.Event]util.Rect)
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
	t.eventviewOffset = 10
	t.eventviewWidth = 80
	t.status = "initial status msg"

  t.Resolution = 12
  t.ScrollOffset = 8 * t.Resolution

	return &t
}

func (t *TUI) SetModel(m *model.Model) {
  t.model = m
}

func (t TUI) DrawText(x, y, w, h int, style tcell.Style, text string) {
	row := y
	col := x
	for _, r := range []rune(text) {
		t.Screen.SetContent(col, row, r, nil, style)
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

func (t TUI) DrawBox(style tcell.Style, x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			t.Screen.SetContent(col, row, ' ', nil, style)
		}
	}
}

func (t TUI) DrawStatus() {
	statusOffset := t.eventviewOffset + t.eventviewWidth + 2
	_, screenHeight := t.Screen.Size()
	t.DrawText(statusOffset, screenHeight-2, 100, 1, tcell.StyleDefault, t.status)
}

func (t TUI) DrawTimeline() {
	_, height := t.Screen.Size()

	now := time.Now()
	h := now.Hour()
	m := now.Minute()
  if t.Resolution == 0 {
    panic("RES IS ZERO?!")
  }
	nowRow := (h * t.Resolution) - t.ScrollOffset + (m / (60 / t.Resolution))

	hour := t.ScrollOffset / t.Resolution

	for row := 0; row <= height; row++ {
		if hour >= 24 {
			break
		}
		style := tcell.StyleDefault.Foreground(tcell.ColorLightGray)
		if row == nowRow {
			style = style.Background(tcell.ColorRed)
		}
		if row%t.Resolution == 0 {
			tStr := fmt.Sprintf("   %s  ", timestamp.Timestamp{Hour: hour, Minute: 0}.ToString())
			t.DrawText(0, row, 10, 1, style, tStr)
			hour++
		} else {
			t.DrawText(0, row, 10, 1, style, "          ")
		}
	}
}

func (t TUI) DrawEvents() {
	selStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	for _, e := range t.model.Events {
		style := t.CategoryStyling[e.Cat]
		// based on event state, draw a box or maybe a smaller one, or ...
		p := t.positions[e]
		if t.Hovered.Event == nil || *t.Hovered.Event != e {
			t.DrawBox(style, p.X, p.Y, p.W, p.H)
			t.DrawText(p.X+1, p.Y, p.W-2, p.H, style, e.Name)
			t.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
			t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
		} else {
			if t.Hovered.Resize {
				t.DrawBox(style, p.X, p.Y, p.W, p.H-1)
				t.DrawBox(selStyle, p.X, p.Y+p.H-1, p.W, 1)
				t.DrawText(p.X+1, p.Y, p.W-2, p.H, style, e.Name)
				t.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			} else {
				t.DrawBox(selStyle, p.X, p.Y, p.W, p.H)
				t.DrawText(p.X+1, p.Y, p.W-2, p.H, selStyle, e.Name)
				t.DrawText(p.X+p.W-5, p.Y, 5, 1, selStyle, e.Start.ToString())
				t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			}
		}
	}
}

func (t TUI) ComputeRects() {
	defaultX := t.eventviewOffset
	defaultW := t.eventviewWidth
	active_stack := make([]model.Event, 0)
	for _, e := range t.model.Events {
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
		for i := 1; i < len(active_stack); i++ {
			x = x + (w / 2)
			w = w / 2
		}
		t.positions[e] = util.Rect{X: x, Y: y, W: w, H: h}
	}
}

func (t TUI) GetHoveredEvent() eventHoverState {
	if t.CursorX >= t.eventviewOffset &&
		t.CursorX < (t.eventviewOffset+t.eventviewWidth) {
		for i := len(t.model.Events) - 1; i >= 0; i-- {
			if t.positions[t.model.Events[i]].Contains(t.CursorX, t.CursorY) {
				if t.CursorY == (t.positions[t.model.Events[i]].Y + t.positions[t.model.Events[i]].H - 1) {
					return eventHoverState{&t.model.Events[i], true}
				} else {
					return eventHoverState{&t.model.Events[i], false}
				}
			}
		}
	}
	return eventHoverState{nil, false}
}

func (t TUI) toY(ts timestamp.Timestamp) int {
	return ((ts.Hour*t.Resolution - t.ScrollOffset) + (ts.Minute / (60 / t.Resolution)))
}

func (t TUI) timeForDistance(dist int) timestamp.TimeOffset {
	add := true
	if dist < 0 {
		dist *= (-1)
		add = false
	}
	minutes := dist * (60 / t.Resolution)
	return timestamp.TimeOffset{T: timestamp.Timestamp{Hour: minutes / 60, Minute: minutes % 60}, Add: add}
}

func (t TUI) EventMove(dist int) {
	e := t.EditedEvent
	timeOffset := t.timeForDistance(dist)
	e.Start = e.Start.Offset(timeOffset).Snap(t.Resolution)
	e.End = e.End.Offset(timeOffset).Snap(t.Resolution)
}
func (t TUI) EventResize(dist int) {
	e := t.EditedEvent
	timeOffset := t.timeForDistance(dist)
	newEnd := e.End.Offset(timeOffset).Snap(t.Resolution)
	if newEnd.IsAfter(e.Start) {
		e.End = newEnd
	}
}
