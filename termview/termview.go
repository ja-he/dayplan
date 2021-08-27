package termview

import (
	"fmt"
	"log"
	"sort"
	"time"

	"dayplan/model"
	"dayplan/timestamp"
	"dayplan/tui"

	"github.com/gdamore/tcell/v2"
)

type editState int

const (
	None editState = iota
	Moving
	Resizing
)

// Initialize the screen checking errors and return it, so long as no critical
// error occurred.
func (t *Termview) initScreen() {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	t.Screen = s
}

type Termview struct {
	Screen      tcell.Screen
	tui         *tui.TUI
	editState   editState
	EditedEvent *model.Event
}

func NewTermview(tui *tui.TUI) *Termview {
	t := Termview{}

	t.initScreen()
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	t.Screen.SetStyle(defStyle)
	t.Screen.EnableMouse()
	t.Screen.EnablePaste()
	t.Screen.Clear()

	t.tui = tui

	return &t
}

func (t Termview) timeForDistance(dist int) timestamp.TimeOffset {
	add := true
	if dist < 0 {
		dist *= (-1)
		add = false
	}
	minutes := dist * (60 / t.tui.Resolution)
	return timestamp.TimeOffset{T: timestamp.Timestamp{Hour: minutes / 60, Minute: minutes % 60}, Add: add}
}

func (t Termview) EventMove(dist int) {
	e := t.EditedEvent
	timeOffset := t.timeForDistance(dist)
	e.Start = e.Start.Offset(timeOffset).Snap(t.tui.Resolution)
	e.End = e.End.Offset(timeOffset).Snap(t.tui.Resolution)
}
func (t Termview) EventResize(dist int) {
	e := t.EditedEvent
	timeOffset := t.timeForDistance(dist)
	newEnd := e.End.Offset(timeOffset).Snap(t.tui.Resolution)
	if newEnd.IsAfter(e.Start) {
		e.End = newEnd
	}
}

func (t Termview) Run() {
	for {
		t.Screen.Show()
		t.Screen.Clear()

		// TODO: this blocks, meaning if no input is given, the screen doesn't update
		//       what we might want is an input buffer in another goroutine? idk
		ev := t.Screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			t.Screen.Sync()
		case *tcell.EventKey:
			// TODO: handle keys
			return
		case *tcell.EventMouse:
			// TODO: handle mouse input
			oldY := t.tui.CursorY
			t.tui.CursorX, t.tui.CursorY = ev.Position()
			button := ev.Buttons()

			if button == tcell.Button1 {
				switch t.editState {
				case None:
					hovered := t.tui.GetHoveredEvent()
					if hovered.Event != nil {
						t.EditedEvent = hovered.Event
						if hovered.Resize {
							t.editState = Resizing
						} else {
							t.editState = Moving
						}
					}
				case Moving:
					t.EditedEvent.Snap(t.tui.Resolution)
					t.EventMove(t.tui.CursorY - oldY)
				case Resizing:
					t.EventResize(t.tui.CursorY - oldY)
				}
			} else if button == tcell.WheelUp {
				newHourOffset := ((t.tui.ScrollOffset / t.tui.Resolution) - 1)
				if newHourOffset >= 0 {
					t.tui.ScrollOffset = newHourOffset * t.tui.Resolution
				}
			} else if button == tcell.WheelDown {
				newHourOffset := ((t.tui.ScrollOffset / t.tui.Resolution) + 1)
				if newHourOffset <= 24 {
					t.tui.ScrollOffset = newHourOffset * t.tui.Resolution
				}
			} else {
				t.editState = None
				sort.Sort(model.ByStart(t.tui.Model.Events))
				t.tui.Hovered = t.tui.GetHoveredEvent()
			}
		}

		t.DrawTimeline()
		t.tui.ComputeRects()
		t.DrawEvents()
		t.DrawStatus()
	}
}

func (t Termview) DrawText(x, y, w, h int, style tcell.Style, text string) {
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

func (t Termview) DrawBox(style tcell.Style, x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			t.Screen.SetContent(col, row, ' ', nil, style)
		}
	}
}

func (t Termview) DrawStatus() {
	statusOffset := t.tui.EventviewOffset + t.tui.EventviewWidth + 2
	_, screenHeight := t.Screen.Size()
	t.DrawText(statusOffset, screenHeight-2, 100, 1, tcell.StyleDefault, t.tui.Status)
}

func (t Termview) DrawTimeline() {
	_, height := t.Screen.Size()

	now := time.Now()
	h := now.Hour()
	m := now.Minute()
	if t.tui.Resolution == 0 {
		panic("RES IS ZERO?!")
	}
	nowRow := (h * t.tui.Resolution) - t.tui.ScrollOffset + (m / (60 / t.tui.Resolution))

	hour := t.tui.ScrollOffset / t.tui.Resolution

	for row := 0; row <= height; row++ {
		if hour >= 24 {
			break
		}
		style := tcell.StyleDefault.Foreground(tcell.ColorLightGray)
		if row == nowRow {
			style = style.Background(tcell.ColorRed)
		}
		if row%t.tui.Resolution == 0 {
			tStr := fmt.Sprintf("   %s  ", timestamp.Timestamp{Hour: hour, Minute: 0}.ToString())
			t.DrawText(0, row, 10, 1, style, tStr)
			hour++
		} else {
			t.DrawText(0, row, 10, 1, style, "          ")
		}
	}
}

func (t Termview) DrawEvents() {
	selStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	for _, e := range t.tui.Model.Events {
		style := t.tui.CategoryStyling[e.Cat]
		// based on event state, draw a box or maybe a smaller one, or ...
		p := t.tui.Positions[e]
		if t.tui.Hovered.Event == nil || *t.tui.Hovered.Event != e {
			t.DrawBox(style, p.X, p.Y, p.W, p.H)
			t.DrawText(p.X+1, p.Y, p.W-2, p.H, style, e.Name)
			t.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
			t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
		} else {
			if t.tui.Hovered.Resize {
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
