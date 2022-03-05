package tui

import (
	"strings"

	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
)

type LogPane struct {
	renderer   ui.ConstrainedRenderer
	dimensions func() (x, y, w, h int)
	stylesheet styling.Stylesheet

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

	if p.condition() {
		x, y, w, h := p.Dimensions()
		row := 2

		p.renderer.DrawBox(x, y, w, h, p.stylesheet.LogDefault())
		title := p.titleString()
		p.renderer.DrawBox(x, y, w, 1, p.stylesheet.LogTitleBox())
		p.renderer.DrawText(x+(w/2-len(title)/2), y, len(title), 1, p.stylesheet.LogTitleBox(), title)
		for i := len(p.logReader.Get()) - 1; i >= 0; i-- {
			entry := &p.logReader.Get()[i]

			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogEntryType(), entry.Type)
			x += len(entry.Type) + 1

			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogDefault(), entry.Message)
			x += len(entry.Message) + 1

			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogEntryLocation(), entry.Location)
			x += len(entry.Location) + 1

			timeStr := strings.Join(strings.Split(entry.At.String(), " ")[0:2], " ")
			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogEntryTime(), timeStr)

			x = 0
			row++
		}
	}
}

func (p *LogPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}
