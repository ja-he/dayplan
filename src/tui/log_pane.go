package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
)

type LogPane struct {
	renderer   ConstrainedRenderer
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
	style := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	if p.condition() {
		x, y, w, h := p.Dimensions()
		row := 2

		p.renderer.DrawBox(x, y, w, h, style)
		title := p.titleString()
		p.renderer.DrawBox(x, y, w, 1, style.Background(tcell.ColorLightGrey))
		p.renderer.DrawText(x+(w/2-len(title)/2), y, len(title), 1, style.Background(tcell.ColorLightGrey).Bold(true), title)
		for i := len(p.logReader.Get()) - 1; i >= 0; i-- {
			entry := &p.logReader.Get()[i]

			p.renderer.DrawText(x, y+row, w, 1, style.Foreground(tcell.ColorDarkGrey).Italic(true), entry.Type)
			x += len(entry.Type) + 1

			p.renderer.DrawText(x, y+row, w, 1, style, entry.Message)
			x += len(entry.Message) + 1

			p.renderer.DrawText(x, y+row, w, 1, style.Foreground(tcell.ColorDarkGrey), entry.Location)
			x += len(entry.Location) + 1

			timeStr := strings.Join(strings.Split(entry.At.String(), " ")[0:2], " ")
			p.renderer.DrawText(x, y+row, w, 1, style.Foreground(tcell.ColorLightGrey), timeStr)

			x = 0
			row++
		}
	}
}

func (p *LogPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}
