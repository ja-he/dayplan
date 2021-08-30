package tui_view

import (
	"fmt"
	"log"
	"time"

	"dayplan/hover_state"
	"dayplan/timestamp"
	"dayplan/tui_model"

	"github.com/gdamore/tcell/v2"
)

type TUIView struct {
	Screen    tcell.Screen
	Model     *tui_model.TUIModel
	needsSync bool
}

func (v *TUIView) NeedsSync() {
	v.needsSync = true
}

// Initialize the screen checking errors and return it, so long as no critical
// error occurred.
func (t *TUIView) initScreen() {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	t.Screen = s
}

func NewTUIView(tui *tui_model.TUIModel) *TUIView {
	t := TUIView{}

	t.initScreen()
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	t.Screen.SetStyle(defStyle)
	t.Screen.EnableMouse()
	t.Screen.EnablePaste()
	t.Screen.Clear()

	t.Model = tui

	return &t
}

func (t TUIView) Render() {

	t.Screen.Clear()
	t.DrawTimeline()
	t.Model.ComputeRects() // TODO: move to controller?
	t.DrawEvents()
	t.DrawStatus()
	if t.needsSync {
		t.needsSync = false
		t.Screen.Sync()
	} else {
		t.Screen.Show()
	}
}

func (t TUIView) DrawText(x, y, w, h int, style tcell.Style, text string) {
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

func (t TUIView) DrawBox(style tcell.Style, x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			t.Screen.SetContent(col, row, ' ', nil, style)
		}
	}
}

func (t TUIView) DrawStatus() {
	statusOffset := t.Model.EventviewOffset + t.Model.EventviewWidth + 2
	_, screenHeight := t.Screen.Size()
	t.DrawText(statusOffset, screenHeight-2, 100, 1, tcell.StyleDefault, t.Model.Status)
}

func (t TUIView) DrawTimeline() {
	_, height := t.Screen.Size()

	now := time.Now()
	h := now.Hour()
	m := now.Minute()
	if t.Model.Resolution == 0 {
		panic("RES IS ZERO?!")
	}
	nowRow := (h * t.Model.Resolution) - t.Model.ScrollOffset + (m / (60 / t.Model.Resolution))

	hour := t.Model.ScrollOffset / t.Model.Resolution

	for row := 0; row <= height; row++ {
		if hour >= 24 {
			break
		}
		style := tcell.StyleDefault.Foreground(tcell.ColorLightGray)
		if row == nowRow {
			style = style.Background(tcell.ColorRed)
		}
		if row%t.Model.Resolution == 0 {
			tStr := fmt.Sprintf("   %s  ", timestamp.Timestamp{Hour: hour, Minute: 0}.ToString())
			t.DrawText(0, row, 10, 1, style, tStr)
			hour++
		} else {
			t.DrawText(0, row, 10, 1, style, "          ")
		}
	}
}

func (t TUIView) DrawEvents() {
	selStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	for _, e := range t.Model.Model.Events {
		style := t.Model.CategoryStyling[e.Cat]
		// based on event state, draw a box or maybe a smaller one, or ...
		p := t.Model.Positions[e.ID]
		if t.Model.Hovered.EventID != e.ID {
			t.DrawBox(style, p.X, p.Y, p.W, p.H)
			t.DrawText(p.X+1, p.Y, p.W-2, p.H, style, e.Name)
			t.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
			t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
		} else {
			switch t.Model.Hovered.HoverState {
			case hover_state.Resize:
				t.DrawBox(style, p.X, p.Y, p.W, p.H-1)
				t.DrawBox(selStyle, p.X, p.Y+p.H-1, p.W, 1)
				t.DrawText(p.X+1, p.Y, p.W-2, p.H, style, e.Name)
				t.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case hover_state.Move:
				t.DrawBox(selStyle, p.X, p.Y, p.W, p.H)
				t.DrawText(p.X+1, p.Y, p.W-2, p.H, selStyle, e.Name)
				t.DrawText(p.X+p.W-5, p.Y, 5, 1, selStyle, e.Start.ToString())
				t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			}
		}
	}
}
