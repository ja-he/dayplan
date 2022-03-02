package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
)

type LogPane struct {
	renderer   ui.ConstrainedRenderer
	dimensions func() (x, y, w, h int)

	logReader potatolog.LogReader

	condition   func() bool
	titleString func() string
}

func (p *LogPane) Condition() bool { return p.condition() }

func (p *LogPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draws the time summary view over top of all previously drawn contents, if it
// is currently active.
func (p *LogPane) Draw() {

	// TODO: these are temporarily still hardcoded, will be moved with
	//       customizable styling being implemented
	defaultStyling := NewStyling(tcell.ColorBlack, tcell.ColorWhite)
	titleBoxStyling := NewStyling(tcell.ColorBlack, tcell.ColorLightGrey).Bolded()
	entryTypeStyling := NewStyling(tcell.ColorDarkGrey, tcell.ColorWhite).Italicized()
	entryLocationStyling := NewStyling(tcell.ColorDarkGrey, tcell.ColorWhite)
	entryTimeStyling := NewStyling(tcell.ColorLightGrey, tcell.ColorWhite)

	if p.condition() {
		x, y, w, h := p.Dimensions()
		row := 2

		p.renderer.DrawBox(x, y, w, h, defaultStyling)
		title := p.titleString()
		p.renderer.DrawBox(x, y, w, 1, titleBoxStyling)
		p.renderer.DrawText(x+(w/2-len(title)/2), y, len(title), 1, titleBoxStyling, title)
		for i := len(p.logReader.Get()) - 1; i >= 0; i-- {
			entry := &p.logReader.Get()[i]

			p.renderer.DrawText(x, y+row, w, 1, entryTypeStyling, entry.Type)
			x += len(entry.Type) + 1

			p.renderer.DrawText(x, y+row, w, 1, defaultStyling, entry.Message)
			x += len(entry.Message) + 1

			p.renderer.DrawText(x, y+row, w, 1, entryLocationStyling, entry.Location)
			x += len(entry.Location) + 1

			timeStr := strings.Join(strings.Split(entry.At.String(), " ")[0:2], " ")
			p.renderer.DrawText(x, y+row, w, 1, entryTimeStyling, timeStr)

			x = 0
			row++
		}
	}
}

func (p *LogPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}
