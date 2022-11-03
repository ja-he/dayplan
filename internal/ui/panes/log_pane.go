package panes

import (
	"sort"

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
			extraDataIndentWidth := levelLen + 1
			p.renderer.DrawText(
				x, y+row, levelLen, 1,
				func() styling.DrawStyling {
					switch entry["level"] {
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
				util.PadCenter(entry["level"], levelLen),
			)
			x = extraDataIndentWidth

			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogDefault, entry["message"])
			x += len(entry["message"]) + 1

			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogEntryLocation, entry["caller"])
			x += len(entry["caller"]) + 1

			timeStr := entry["time"]
			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogEntryTime, timeStr)

			x = extraDataIndentWidth
			row++

			keys := make([]string, len(entry))
			i := 0
			for k := range entry {
				keys[i] = k
				i++
			}
			sort.Sort(ByAlphabeticOrder(keys))
			for _, k := range keys {
				if k != "caller" && k != "message" && k != "time" && k != "level" {
					p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogEntryTime, k)
					p.renderer.DrawText(x+len(k)+2, y+row, w, 1, p.stylesheet.LogEntryLocation, entry[k])
					row++
				}
			}

			x = 0
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

type ByAlphabeticOrder []string

func (a ByAlphabeticOrder) Len() int           { return len(a) }
func (a ByAlphabeticOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAlphabeticOrder) Less(i, j int) bool { return a[i] < a[j] }
