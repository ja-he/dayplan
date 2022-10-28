package panes

import (
	"strings"

	"github.com/ja-he/dayplan/internal/potatolog"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
)

// LogPane shows the log, with the most recent log entries at the top.
type LogPane struct {
	Leaf

	logReader potatolog.LogReader

	titleString func() string
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *LogPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw draws the time summary view over top of all previously drawn contents,
// if it is currently active.
func (p *LogPane) Draw() {

	if p.IsVisible() {
		x, y, w, h := p.Dimensions()
		row := 2

		p.renderer.DrawBox(x, y, w, h, p.stylesheet.LogDefault)
		title := p.titleString()
		p.renderer.DrawBox(x, y, w, 1, p.stylesheet.LogTitleBox)
		p.renderer.DrawText(x+(w/2-len(title)/2), y, len(title), 1, p.stylesheet.LogTitleBox, title)
		for i := len(p.logReader.Get()) - 1; i >= 0; i-- {
			entry := p.logReader.Get()[i]

			levelLen := len(" error ")
			p.renderer.DrawText(
				x, y+row, levelLen, 1,
				func() styling.DrawStyling {
					switch entry.Level {
					case "error":
						return p.stylesheet.LogEntryTypeError
					case "warn":
						return p.stylesheet.LogEntryTypeWarn
					case "info":
						return p.stylesheet.LogEntryTypeInfo
					case "debug":
						return p.stylesheet.LogEntryTypeDebug
					case "trace":
						return p.stylesheet.LogEntryTypeTrace
					}
					return p.stylesheet.LogDefault
				}(),
				util.PadCenter(entry.Level, levelLen),
			)
			x += levelLen + 1

			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogDefault, entry.Message)
			x += len(entry.Message) + 1

			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogEntryLocation, entry.Caller)
			x += len(entry.Caller) + 1

			timeStr := strings.Join(strings.Split(entry.Time.String(), " ")[0:2], " ")
			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogEntryTime, timeStr)

			x = 0
			row++
		}
	}
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *LogPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}

// NewLogPane constructs and returns a new LogPane.
func NewLogPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	condition func() bool,
	titleString func() string,
	logReader potatolog.LogReader,
) *LogPane {
	return &LogPane{
		Leaf: Leaf{
			Base: Base{
				Visible: condition,
				ID:      ui.GeneratePaneID(),
			},
			renderer:   renderer,
			dimensions: dimensions,
			stylesheet: stylesheet,
		},
		titleString: titleString,
		logReader:   logReader,
	}
}
