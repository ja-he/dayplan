package main

import (
	"bufio"
	"log"
	"os"
	"sort"

	"dayplan/model"
	"dayplan/tui"

	"github.com/gdamore/tcell/v2"
)

// MAIN
func main() {
	filename := os.Args[1]
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot read file '%s'", filename)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var m model.Model
	for scanner.Scan() {
		s := scanner.Text()
		m.AddEvent(*model.NewEvent(s))
	}

	tv := tui.NewTUI()
  defer tv.Screen.Fini()
	tv.SetModel(&m)

	for {
		tv.Screen.Show()
		tv.Screen.Clear()

		// TODO: this blocks, meaning if no input is given, the screen doesn't update
		//       what we might want is an input buffer in another goroutine? idk
		ev := tv.Screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			tv.Screen.Sync()
		case *tcell.EventKey:
			// TODO: handle keys
			tv.Screen.Fini()
			os.Exit(0)
		case *tcell.EventMouse:
			// TODO: handle mouse input
			oldY := tv.CursorY
			tv.CursorX, tv.CursorY = ev.Position()
			button := ev.Buttons()

			if button == tcell.Button1 {
				switch tv.EditState {
				case tui.None:
					hovered := tv.GetHoveredEvent()
					if hovered.Event != nil {
						tv.EditedEvent = hovered.Event
						if hovered.Resize {
							tv.EditState = tui.Resizing
						} else {
							tv.EditState = tui.Moving
						}
					}
				case tui.Moving:
					tv.EditedEvent.Snap(tv.Resolution)
					tv.EventMove(tv.CursorY - oldY)
				case tui.Resizing:
					tv.EventResize(tv.CursorY - oldY)
				}
			} else if button == tcell.WheelUp {
				newHourOffset := ((tv.ScrollOffset / tv.Resolution) - 1)
				if newHourOffset >= 0 {
					tv.ScrollOffset = newHourOffset * tv.Resolution
				}
			} else if button == tcell.WheelDown {
				newHourOffset := ((tv.ScrollOffset / tv.Resolution) + 1)
				if newHourOffset <= 24 {
					tv.ScrollOffset = newHourOffset * tv.Resolution
				}
			} else {
				tv.EditState = tui.None
				sort.Sort(model.ByStart(m.Events))
				tv.Hovered = tv.GetHoveredEvent()
			}
		}

		tv.DrawTimeline()
		tv.ComputeRects()
		tv.DrawEvents()
		tv.DrawStatus()
	}
}
