package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"dayplan/model"
	"dayplan/timestamp"

	"github.com/gdamore/tcell/v2"
)

var resolution = 12
var scrollOffset = 8 * resolution

func timeForDistance(dist int) timestamp.TimeOffset {
	add := true
	if dist < 0 {
		dist *= (-1)
		add = false
	}
	minutes := dist * (60 / resolution)
	return timestamp.TimeOffset{T: timestamp.Timestamp{Hour: minutes / 60, Minute: minutes % 60}, Add: add}
}
func eventMove(e *model.Event, dist int) {
	timeOffset := timeForDistance(dist)
	e.Start = e.Start.Offset(timeOffset).Snap(resolution)
	e.End = e.End.Offset(timeOffset).Snap(resolution)
}
func eventResize(e *model.Event, dist int) {
	timeOffset := timeForDistance(dist)
	newEnd := e.End.Offset(timeOffset).Snap(resolution)
	if newEnd.IsAfter(e.Start) {
		e.End = newEnd
	}
}

// TODO: make this a TUI method
func toY(t timestamp.Timestamp) int {
	return ((t.Hour*resolution - scrollOffset) + (t.Minute / (60 / resolution)))
}

type rect struct {
	x, y, w, h int
}

func within(x, y int, r rect) bool {
	return (x >= r.x) && (x < r.x+r.w) &&
		(y >= r.y) && (y < r.y+r.h)
}

type editState int

const (
	none editState = iota
	moving
	resizing
)

type eventHoverState struct {
	e      *model.Event
	resize bool
}
type TUI struct {
	scaleFactor                     int
	scrollOffset                    int
	screen                          tcell.Screen
	cursorX, cursorY                int
	eventviewOffset, eventviewWidth int
	categoryStyling                 map[model.Category]tcell.Style
	positions                       map[model.Event]rect
	hovered                         eventHoverState
	editState                       editState
	editedEvent                     *model.Event
	status                          string
}

func drawText(s tcell.Screen, x, y, w, h int, style tcell.Style, text string) {
	row := y
	col := x
	for _, r := range []rune(text) {
		s.SetContent(col, row, r, nil, style)
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
func drawTimeline(s tcell.Screen) {
	_, height := s.Size()

	now := time.Now()
	h := now.Hour()
	m := now.Minute()
	nowRow := (h * resolution) - scrollOffset + (m / (60 / resolution))

	hour := scrollOffset / resolution

	for row := 0; row <= height; row++ {
		if hour >= 24 {
			break
		}
		style := tcell.StyleDefault.Foreground(tcell.ColorLightGray)
		if row == nowRow {
			style = style.Background(tcell.ColorRed)
		}
		if row%resolution == 0 {
			tStr := fmt.Sprintf("   %s  ", timestamp.Timestamp{hour, 0}.ToString())
			drawText(s, 0, row, 10, 1, style, tStr)
			hour++
		} else {
			drawText(s, 0, row, 10, 1, style, "          ")
		}
	}
}
func drawBox(screen tcell.Screen, style tcell.Style, x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			screen.SetContent(col, row, ' ', nil, style)
		}
	}
}

func drawStatus(tv TUI) {
	statusOffset := tv.eventviewOffset + tv.eventviewWidth + 2
	_, screenHeight := tv.screen.Size()
	drawText(tv.screen, statusOffset, screenHeight-2, 100, 1, tcell.StyleDefault, tv.status)
}

func drawEvents(tv TUI, m model.Model) {
	selStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	for _, e := range m.Events {
		style := tv.categoryStyling[e.Cat]
		// based on event state, draw a box or maybe a smaller one, or ...
		p := tv.positions[e]
		if tv.hovered.e == nil || *tv.hovered.e != e {
			drawBox(tv.screen, style, p.x, p.y, p.w, p.h)
			drawText(tv.screen, p.x+1, p.y, p.w-2, p.h, style, e.Name)
			drawText(tv.screen, p.x+p.w-5, p.y, 5, 1, style, e.Start.ToString())
			drawText(tv.screen, p.x+p.w-5, p.y+p.h-1, 5, 1, style, e.End.ToString())
		} else {
			if tv.hovered.resize {
				drawBox(tv.screen, style, p.x, p.y, p.w, p.h-1)
				drawBox(tv.screen, selStyle, p.x, p.y+p.h-1, p.w, 1)
				drawText(tv.screen, p.x+1, p.y, p.w-2, p.h, style, e.Name)
				drawText(tv.screen, p.x+p.w-5, p.y, 5, 1, style, e.Start.ToString())
				drawText(tv.screen, p.x+p.w-5, p.y+p.h-1, 5, 1, selStyle, e.End.ToString())
			} else {
				drawBox(tv.screen, selStyle, p.x, p.y, p.w, p.h)
				drawText(tv.screen, p.x+1, p.y, p.w-2, p.h, selStyle, e.Name)
				drawText(tv.screen, p.x+p.w-5, p.y, 5, 1, selStyle, e.Start.ToString())
				drawText(tv.screen, p.x+p.w-5, p.y+p.h-1, 5, 1, selStyle, e.End.ToString())
			}
		}
	}
}

func computeRects(tv TUI, m model.Model) {
	defaultX := tv.eventviewOffset
	defaultW := tv.eventviewWidth
	active_stack := make([]model.Event, 0)
	for _, e := range m.Events {
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
		y := toY(e.Start)
		x := defaultX
		h := toY(e.End) - y
		w := defaultW
		for i := 1; i < len(active_stack); i++ {
			x = x + (w / 2)
			w = w / 2
		}
		tv.positions[e] = rect{x, y, w, h}
	}
}

func getHoveredEvent(tv TUI, m model.Model) eventHoverState {
	if tv.cursorX >= tv.eventviewOffset &&
		tv.cursorX < (tv.eventviewOffset+tv.eventviewWidth) {
		for i := len(m.Events) - 1; i >= 0; i-- {
			if within(tv.cursorX, tv.cursorY, tv.positions[m.Events[i]]) {
				if tv.cursorY == (tv.positions[m.Events[i]].y + tv.positions[m.Events[i]].h - 1) {
					return eventHoverState{&m.Events[i], true}
				} else {
					return eventHoverState{&m.Events[i], false}
				}
			}
		}
	}
	return eventHoverState{nil, false}
}

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

// MAIN
func main() {
	filename := os.Args[1]
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot read file '%s'", filename)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)

	var tv TUI

	tv.screen = initScreen()
	tv.screen.SetStyle(defStyle)
	tv.screen.EnableMouse()
	tv.screen.EnablePaste()
	tv.screen.Clear()
	tv.categoryStyling = make(map[model.Category]tcell.Style)
	tv.positions = make(map[model.Event]rect)
	tv.categoryStyling[model.Category{Name: "work"}] = tcell.StyleDefault.Background(tcell.NewHexColor(0xccebff)).Foreground(tcell.ColorReset)
	tv.categoryStyling[model.Category{Name: "leisure"}] = tcell.StyleDefault.Background(tcell.Color76).Foreground(tcell.ColorReset)
	tv.categoryStyling[model.Category{Name: "misc"}] = tcell.StyleDefault.Background(tcell.Color250).Foreground(tcell.ColorReset)
	tv.categoryStyling[model.Category{Name: "programming"}] = tcell.StyleDefault.Background(tcell.Color226).Foreground(tcell.ColorReset)
	tv.categoryStyling[model.Category{Name: "cooking"}] = tcell.StyleDefault.Background(tcell.Color212).Foreground(tcell.ColorReset)
	tv.categoryStyling[model.Category{Name: "fitness"}] = tcell.StyleDefault.Background(tcell.Color208).Foreground(tcell.ColorReset)
	tv.categoryStyling[model.Category{Name: "eating"}] = tcell.StyleDefault.Background(tcell.Color224).Foreground(tcell.ColorReset)
	tv.categoryStyling[model.Category{Name: "hygiene"}] = tcell.StyleDefault.Background(tcell.Color80).Foreground(tcell.ColorReset)
	tv.categoryStyling[model.Category{Name: "cleaning"}] = tcell.StyleDefault.Background(tcell.Color215).Foreground(tcell.ColorReset)
	tv.categoryStyling[model.Category{Name: "laundry"}] = tcell.StyleDefault.Background(tcell.Color111).Foreground(tcell.ColorReset)
	tv.categoryStyling[model.Category{Name: "family"}] = tcell.StyleDefault.Background(tcell.Color122).Foreground(tcell.ColorReset)
	tv.eventviewOffset = 10
	tv.eventviewWidth = 80
	tv.status = "initial status msg"

	defer tv.screen.Fini()

	var m model.Model
	for scanner.Scan() {
		s := scanner.Text()
		m.AddEvent(*model.NewEvent(s))
	}

	for {
		tv.screen.Show()
		tv.screen.Clear()

		// TODO: this blocks, meaning if no input is given, the screen doesn't update
		//       what we might want is an input buffer in another goroutine? idk
		ev := tv.screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			tv.screen.Sync()
		case *tcell.EventKey:
			// TODO: handle keys
			tv.screen.Fini()
			os.Exit(0)
		case *tcell.EventMouse:
			// TODO: handle mouse input
			oldY := tv.cursorY
			tv.cursorX, tv.cursorY = ev.Position()
			button := ev.Buttons()

			if button == tcell.Button1 {
				switch tv.editState {
				case none:
					hovered := getHoveredEvent(tv, m)
					if hovered.e != nil {
						tv.editedEvent = hovered.e
						if hovered.resize {
							tv.editState = resizing
						} else {
							tv.editState = moving
						}
					}
				case moving:
					eventMove(tv.editedEvent, tv.cursorY-oldY)
				case resizing:
					eventResize(tv.editedEvent, tv.cursorY-oldY)
				}
			} else if button == tcell.WheelUp {
				newHourOffset := ((scrollOffset / resolution) - 1)
				if newHourOffset >= 0 {
					scrollOffset = newHourOffset * resolution
				}
			} else if button == tcell.WheelDown {
				newHourOffset := ((scrollOffset / resolution) + 1)
				if newHourOffset <= 24 {
					scrollOffset = newHourOffset * resolution
				}
			} else {
				tv.editState = none
				sort.Sort(model.ByStart(m.Events))
				tv.hovered = getHoveredEvent(tv, m)
			}
		}

		drawTimeline(tv.screen)
		computeRects(tv, m)
		drawEvents(tv, m)
		drawStatus(tv)
	}
}
