package panes

import (
	"fmt"
	"sort"

	"github.com/ja-he/dayplan/internal/potatolog"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
)

// LogPane shows the log, with the most recent log entries at the top.
type LogPane struct {
	ui.LeafPane

	logReader potatolog.LogReader

	titleString func() string
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *LogPane) Dimensions() (x, y, w, h int) {
	return p.Dims()
}

// Draw draws the time summary view over top of all previously drawn contents,
// if it is currently active.
func (p *LogPane) Draw() {

	if p.IsVisible() {
		x, y, w, h := p.Dimensions()
		row := 2

		p.Renderer.DrawBox(x, y, w, h, p.Stylesheet.LogDefault)
		title := p.titleString()
		p.Renderer.DrawBox(x, y, w, 1, p.Stylesheet.LogTitleBox)
		p.Renderer.DrawText(x+(w/2-len(title)/2), y, len(title), 1, p.Stylesheet.LogTitleBox, title)
		for i := len(p.logReader.Get()) - 1; i >= 0; i-- {
			entry := p.logReader.Get()[i]

			levelLen := len(" error ")
			extraDataIndentWidth := levelLen + 1
			p.Renderer.DrawText(
				x, y+row, levelLen, 1,
				func() styling.DrawStyling {
					switch entry["level"] {
					case "error":
						return p.Stylesheet.LogEntryTypeError
					case "warn":
						return p.Stylesheet.LogEntryTypeWarn
					case "info":
						return p.Stylesheet.LogEntryTypeInfo
					case "debug":
						return p.Stylesheet.LogEntryTypeDebug
					case "trace":
						return p.Stylesheet.LogEntryTypeTrace
					}
					return p.Stylesheet.LogDefault
				}(),
				util.PadCenter(fmt.Sprint(entry["level"]), levelLen),
			)
			x = extraDataIndentWidth

			p.Renderer.DrawText(x, y+row, w, 1, p.Stylesheet.LogDefault, fmt.Sprint(entry["message"]))
			x += len(fmt.Sprint(entry["message"])) + 1

			p.Renderer.DrawText(x, y+row, w, 1, p.Stylesheet.LogEntryLocation, fmt.Sprint(entry["caller"]))
			x += len(fmt.Sprint(entry["caller"])) + 1

			timeStr := fmt.Sprint(entry["time"])
			p.Renderer.DrawText(x, y+row, w, 1, p.Stylesheet.LogEntryTime, timeStr)

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
					p.Renderer.DrawText(x, y+row, w, 1, p.Stylesheet.LogEntryTime, k)
					p.Renderer.DrawText(x+len(k)+2, y+row, w, 1, p.Stylesheet.LogEntryLocation, fmt.Sprint(entry[k]))
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
		LeafPane: ui.LeafPane{
			BasePane: ui.BasePane{
				Visible: condition,
				ID:      ui.GeneratePaneID(),
			},
			Renderer:   renderer,
			Dims:       dimensions,
			Stylesheet: stylesheet,
		},
		titleString: titleString,
		logReader:   logReader,
	}
}

type ByAlphabeticOrder []string

func (a ByAlphabeticOrder) Len() int           { return len(a) }
func (a ByAlphabeticOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAlphabeticOrder) Less(i, j int) bool { return a[i] < a[j] }
