package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gdamore/tcell/v2"
)

type timestamp struct {
	hour   int
	minute int
}

func toString(t timestamp) string {
	return fmt.Sprintf("%d:%d", t.hour, t.minute)
}

func fromY(y int) timestamp {
	t := timestamp{}
	t.hour = (y / 8) + 8
	t.minute = (y % 8) * (60 / 8)
	return t
}

func toY(t timestamp) int {
	y := ((t.hour - 8) * 8) + (t.minute / (60 / 8))
	return y
}

type event struct {
	name  string
	start timestamp
	end   timestamp
}

type styledEvent struct {
	event event
	style tcell.Style
}

func drawText(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {
	row := y1
	col := x1
	for _, r := range []rune(text) {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

func drawStatus(s tcell.Screen, style tcell.Style, status string) {
	x1 := 95
	x2 := 120
	y1 := 0
	y2 := 5

	// Fill background
	for row := y1; row <= y2; row++ {
		for col := x1; col <= x2; col++ {
			s.SetContent(col, row, ' ', nil, style)
		}
	}

	drawText(s, x1, y1, x2, y2, style, status)
}

func drawEventviewColorbar(s tcell.Screen, style tcell.Style, y int) {
	x1 := 10
	x2 := 90

	// Fill background
	for col := x1; col <= x2; col++ {
		s.SetContent(col, y, ' ', nil, style)
	}
}

func drawEvent(s tcell.Screen, style tcell.Style, e event) {
	x1 := 10
	x2 := 90
	y1 := toY(e.start)
	y2 := toY(e.end) - 1

	// Fill background
	for row := y1; row <= y2; row++ {
		for col := x1; col <= x2; col++ {
			s.SetContent(col, row, ' ', nil, style)
		}
	}

	drawText(s, x1, y1, x2, y2, style, e.name)
}

func drawTimeline(s tcell.Screen, style tcell.Style) {
	_, height := s.Size()

	hour := 8

	for row := 0; row <= height; row++ {
		if row%8 == 0 {
			pad := ""
			if hour < 10 {
				pad = " "
			}
			h := fmt.Sprintf("%s%d:00 - ", pad, hour)
			drawText(s, 1, row, 9, row, style, fmt.Sprintf("%s - ", h))
			hour++
		}
	}
}

func overEvent(cursorY int, e event) bool {
	return cursorY >= toY(e.start) && cursorY <= (toY(e.end)-1)
}

type model struct {
	events []styledEvent
}

type view struct {
	zoom             int
	scroll           int
	cursorX, cursorY int
	selected         int
	clicked          bool
	resize           bool
	model            model
}

func main() {
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorReset).Background(tcell.Color225)
	boxStyle2 := tcell.StyleDefault.Foreground(tcell.ColorReset).Background(tcell.Color195)
	selStyle := tcell.StyleDefault.Foreground(tcell.ColorReset).Background(tcell.Color226)
	timelineStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGray).Background(tcell.ColorReset)

	// Initialize screen
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.EnablePaste()
	s.Clear()

	v := view{2, 0, -1, -1, -1, false, false, model{}}

	v.model.events = append(v.model.events, styledEvent{event{"hello", timestamp{8, 15}, timestamp{8, 45}}, boxStyle})
	v.model.events = append(v.model.events, styledEvent{event{"lmao", timestamp{9, 00}, timestamp{10, 00}}, boxStyle2})
	v.model.events = append(v.model.events, styledEvent{event{"third event", timestamp{10, 00}, timestamp{12, 00}}, boxStyle})
	v.model.events = append(v.model.events, styledEvent{event{"very short event", timestamp{14, 00}, timestamp{14, 45}}, boxStyle})

	// Event loop
	quit := func() {
		s.Fini()
		os.Exit(0)
	}

	status := ""

	for {
		// Update screen
		s.Show()
    s.Clear()

		drawTimeline(s, timelineStyle)

		for _, e := range v.model.events {
			drawEvent(s, e.style, e.event)
		}

		// Poll event
		ev := s.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				quit()
			} else if ev.Key() == tcell.KeyCtrlL {
				s.Sync()
			} else if ev.Rune() == 'C' || ev.Rune() == 'c' {
				s.Clear()
			}
		case *tcell.EventMouse:
			v.cursorX, v.cursorY = ev.Position()
			status = fmt.Sprintf("%d , %d", v.cursorX, v.cursorY)

			if !v.clicked {
				if v.cursorX < 10 {
					status += " (TL)"
					v.selected = -1
				} else if v.cursorX < 90 {
					status += " (EL:"
					v.selected = -1
					for i, e := range v.model.events {
						if overEvent(v.cursorY, e.event) {
							v.selected = i
							status += e.event.name
							if v.cursorY == toY(e.event.end)-1 {
								v.resize = true
								drawEventviewColorbar(s, selStyle, v.cursorY)
							} else {
								v.resize = false
								drawEvent(s, selStyle, e.event)
							}
						}
					}
					status += ")"
				} else {
					v.selected = -1
				}
			}

			button := ev.Buttons()
			// Only process button events, not wheel events
			button &= tcell.ButtonMask(0xff)

			if button == tcell.Button1 {
				v.clicked = true
				status += " [BUTTON1]"
				status += " resize to time "
				status += toString(fromY(v.cursorY))
				status += fmt.Sprint(" ", v.selected)
				if v.resize {
					v.model.events[v.selected].event.end = fromY(v.cursorY)
				}
			} else {
				v.clicked = false
			}

		}

		drawStatus(s, defStyle, status)

	}
}
