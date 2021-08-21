package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var resolution = 6
var scrollOffset = 8 * resolution

type timestamp struct {
	hour, minute int
}

func newTimestamp(s string) *timestamp {
	components := strings.Split(s, ":")
	if len(components) != 2 {
		log.Fatalf("given string '%s' which does not fit the HH:MM format", s)
	}
	hStr := components[0]
	mStr := components[1]
	if len(hStr) != 2 || len(mStr) != 2 {
		log.Fatalf("given string '%s' which does not fit the HH:MM format", s)
	}
	h, err := strconv.Atoi(hStr)
	if err != nil {
		log.Fatalf("error converting hour string '%s' to a number", hStr)
	}
	m, err := strconv.Atoi(mStr)
	if err != nil {
		log.Fatalf("error converting minute string '%s' to a number", mStr)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		log.Fatalf("error with string-to-timestamp conversion: one of the yielded values illegal (%d) (%d)", h, m)
	}
	return &timestamp{h, m}
}

func (a timestamp) toString() string {
	hPrefix := ""
	mPrefix := ""
	if a.hour < 10 {
		hPrefix = "0"
	}
	if a.minute < 10 {
		mPrefix = "0"
	}
	return fmt.Sprintf("%s%d:%s%d", hPrefix, a.hour, mPrefix, a.minute)
}

type timeOffset struct {
	t   timestamp
	add bool
}

func (t timestamp) snap(res int) timestamp {
	closestMinute := 0
	for i := 0; i <= 60; i += (60 / res) {
		distance := math.Abs(float64(i - t.minute))
		if distance < math.Abs(float64(closestMinute-t.minute)) {
			closestMinute = i
		}
	}
	if closestMinute == 60 {
		t.hour += 1
		t.minute = 0
	} else {
		t.minute = closestMinute
	}
	return t
}

func (t timestamp) offset(o timeOffset) timestamp {
	if o.add {
		t.hour += o.t.hour
		t.minute += o.t.minute
		if t.minute >= 60 {
			t.minute %= 60
			t.hour += 1
		}
	} else {
		t.minute -= o.t.minute
		t.hour -= o.t.hour
		if t.minute < 0 {
			t.minute = 60 + t.minute
			t.hour -= 1
		}
	}
	return t
}

func timeForDistance(dist int) timeOffset {
	add := true
	if dist < 0 {
		dist *= (-1)
		add = false
	}
	minutes := dist * (60 / resolution)
	return timeOffset{timestamp{minutes / 60, minutes % 60}, add}
}
func toY(t timestamp) int {
	return ((t.hour*resolution - scrollOffset) + (t.minute / (60 / resolution)))
}
func past(a, b timestamp) bool {
	if a.hour > b.hour {
		return true
	} else if a.hour == b.hour {
		return a.minute > b.minute
	} else {
		return false
	}
}

type category struct {
	name string
}
type event struct {
	start, end timestamp
	name       string
	cat        category
}

func newEvent(s string) *event {
	var e event

	args := strings.SplitN(s, "|", 4)
	startString := args[0]
	endString := args[1]
	catString := args[2]
	nameString := args[3]

	e.start = *newTimestamp(startString)
	e.end = *newTimestamp(endString)

	e.name = nameString
	e.cat.name = catString

	return &e
}

func eventMove(e *event, dist int) {
	timeOffset := timeForDistance(dist)
	e.start = e.start.offset(timeOffset).snap(resolution)
	e.end = e.end.offset(timeOffset).snap(resolution)
}
func eventResize(e *event, dist int) {
	timeOffset := timeForDistance(dist)
	newEnd := e.end.offset(timeOffset).snap(resolution)
	if past(newEnd, e.start) {
		e.end = newEnd
	}
}

type ByStart []event

func (a ByStart) Len() int           { return len(a) }
func (a ByStart) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByStart) Less(i, j int) bool { return past(a[j].start, a[i].start) }

type model struct {
	events []event
}

func (m *model) addEvent(e event) {
	m.events = append(m.events, e)
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
	e      *event
	resize bool
}
type termview struct {
	scaleFactor                     int
	scrollOffset                    int
	screen                          tcell.Screen
	cursorX, cursorY                int
	eventviewOffset, eventviewWidth int
	categoryStyling                 map[category]tcell.Style
	positions                       map[event]rect
	hovered                         eventHoverState
	editState                       editState
	editedEvent                     *event
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

	hour := scrollOffset / resolution

	for row := 0; row <= height; row++ {
		if row%resolution == 0 {
			tStr := timestamp{hour, 0}.toString()
			drawText(s, 1, row, 8, 1, tcell.StyleDefault.Foreground(tcell.ColorLightGray), tStr)
			hour++
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

func drawStatus(tv termview) {
	statusOffset := tv.eventviewOffset + tv.eventviewWidth + 2
	_, screenHeight := tv.screen.Size()
	drawText(tv.screen, statusOffset, screenHeight-2, 100, 1, tcell.StyleDefault, tv.status)
}

func drawEvents(tv termview, m model) {
	selStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	for _, e := range m.events {
		style := tv.categoryStyling[e.cat]
		// based on event state, draw a box or maybe a smaller one, or ...
		p := tv.positions[e]
		if tv.hovered.e == nil || *tv.hovered.e != e {
			drawBox(tv.screen, style, p.x, p.y, p.w, p.h)
			drawText(tv.screen, p.x, p.y, p.w, p.h, style, e.name)
			drawText(tv.screen, p.x+p.w-5, p.y, 5, 1, style, e.start.toString())
			drawText(tv.screen, p.x+p.w-5, p.y+p.h-1, 5, 1, style, e.end.toString())
		} else {
			if tv.hovered.resize {
				drawBox(tv.screen, style, p.x, p.y, p.w, p.h-1)
				drawBox(tv.screen, selStyle, p.x, p.y+p.h-1, p.w, 1)
				drawText(tv.screen, p.x, p.y, p.w, p.h, style, e.name)
				drawText(tv.screen, p.x+p.w-5, p.y, 5, 1, style, e.start.toString())
				drawText(tv.screen, p.x+p.w-5, p.y+p.h-1, 5, 1, selStyle, e.end.toString())
			} else {
				drawBox(tv.screen, selStyle, p.x, p.y, p.w, p.h)
				drawText(tv.screen, p.x, p.y, p.w, p.h, selStyle, e.name)
				drawText(tv.screen, p.x+p.w-5, p.y, 5, 1, selStyle, e.start.toString())
				drawText(tv.screen, p.x+p.w-5, p.y+p.h-1, 5, 1, selStyle, e.end.toString())
			}
		}
	}
}

func computeRects(tv termview, m model) {
	defaultX := tv.eventviewOffset
	defaultW := tv.eventviewWidth
	active_stack := make([]event, 0)
	for _, e := range m.events {
		// remove all stacked elements that have finished
		for i := len(active_stack) - 1; i >= 0; i-- {
			if past(e.start, active_stack[i].end) || e.start == active_stack[i].end {
				active_stack = active_stack[:i]
			} else {
				break
			}
		}
		active_stack = append(active_stack, e)
		// based on event state, draw a box or maybe a smaller one, or ...
		y := toY(e.start)
		x := defaultX
		h := toY(e.end) - y
		w := defaultW
		for i := 1; i < len(active_stack); i++ {
			x = x + (w / 2)
			w = w / 2
		}
		tv.positions[e] = rect{x, y, w, h}
	}
}

func getHoveredEvent(tv termview, m model) eventHoverState {
	if tv.cursorX >= tv.eventviewOffset &&
		tv.cursorX < (tv.eventviewOffset+tv.eventviewWidth) {
		for i := len(m.events) - 1; i >= 0; i-- {
			if within(tv.cursorX, tv.cursorY, tv.positions[m.events[i]]) {
				if tv.cursorY == (tv.positions[m.events[i]].y + tv.positions[m.events[i]].h - 1) {
					return eventHoverState{&m.events[i], true}
				} else {
					return eventHoverState{&m.events[i], false}
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

	var tv termview

	tv.screen = initScreen()
	tv.screen.SetStyle(defStyle)
	tv.screen.EnableMouse()
	tv.screen.EnablePaste()
	tv.screen.Clear()
	tv.categoryStyling = make(map[category]tcell.Style)
	tv.positions = make(map[event]rect)
	tv.categoryStyling[category{"work"}]        = tcell.StyleDefault.Background(tcell.Color196).Foreground(tcell.ColorReset)
	tv.categoryStyling[category{"leisure"}]     = tcell.StyleDefault.Background(tcell.Color76).Foreground(tcell.ColorReset)
	tv.categoryStyling[category{"misc"}]        = tcell.StyleDefault.Background(tcell.Color250).Foreground(tcell.ColorReset)
	tv.categoryStyling[category{"programming"}] = tcell.StyleDefault.Background(tcell.Color226).Foreground(tcell.ColorReset)
	tv.categoryStyling[category{"cooking"}]     = tcell.StyleDefault.Background(tcell.Color212).Foreground(tcell.ColorReset)
	tv.categoryStyling[category{"eating"}]      = tcell.StyleDefault.Background(tcell.Color224).Foreground(tcell.ColorReset)
	tv.eventviewOffset = 10
	tv.eventviewWidth = 80
	tv.status = "initial status msg"

	defer tv.screen.Fini()

	var m model
	for scanner.Scan() {
		s := scanner.Text()
		m.addEvent(*newEvent(s))
	}

	for {
		tv.screen.Show()
		tv.screen.Clear()

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
			} else {
				tv.editState = none
				sort.Sort(ByStart(m.events))
				tv.hovered = getHoveredEvent(tv, m)
			}

		}

		drawTimeline(tv.screen)
		computeRects(tv, m)
		drawEvents(tv, m)
		drawStatus(tv)
	}
}
