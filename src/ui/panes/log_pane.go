package panes

import (
	"strings"

	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
)

// LogPane shows the log, with the most recent log entries at the top.
type LogPane struct {
	renderer   ui.ConstrainedRenderer
	dimensions func() (x, y, w, h int)
	stylesheet styling.Stylesheet

	logReader potatolog.LogReader

	Parent         ui.FocusQueriable
	inputProcessor input.ModalInputProcessor

	condition   func() bool
	titleString func() string
}

// EnsureHidden informs the pane that it is not being shown so that it can take
// potential actions to ensure that, e.g., hide the terminal cursor, if
// necessary.
func (p *LogPane) EnsureHidden() {}

// Condition returns whether this pane should be visible.
func (p *LogPane) Condition() bool { return p.condition() }

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *LogPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw draws the time summary view over top of all previously drawn contents,
// if it is currently active.
func (p *LogPane) Draw() {

	if p.condition() {
		x, y, w, h := p.Dimensions()
		row := 2

		p.renderer.DrawBox(x, y, w, h, p.stylesheet.LogDefault)
		title := p.titleString()
		p.renderer.DrawBox(x, y, w, 1, p.stylesheet.LogTitleBox)
		p.renderer.DrawText(x+(w/2-len(title)/2), y, len(title), 1, p.stylesheet.LogTitleBox, title)
		for i := len(p.logReader.Get()) - 1; i >= 0; i-- {
			entry := &p.logReader.Get()[i]

			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogEntryType, entry.Type)
			x += len(entry.Type) + 1

			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogDefault, entry.Message)
			x += len(entry.Message) + 1

			p.renderer.DrawText(x, y+row, w, 1, p.stylesheet.LogEntryLocation, entry.Location)
			x += len(entry.Location) + 1

			timeStr := strings.Join(strings.Split(entry.At.String(), " ")[0:2], " ")
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

func (p *LogPane) CapturesInput() bool {
	return p.inputProcessor != nil && p.inputProcessor.CapturesInput()
}
func (p *LogPane) ProcessInput(key input.Key) bool {
	return p.inputProcessor != nil && p.inputProcessor.ProcessInput(key)
}

func (p *LogPane) HasFocus() bool              { return p.Parent.HasFocus() && p.Parent.Focusses() == p }
func (p *LogPane) Focusses() ui.FocussablePane { return nil }

func (p *LogPane) SetParent(parent ui.FocusQueriable) {
	p.Parent = parent
}

func (p *LogPane) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index int) {
	return p.inputProcessor.ApplyModalOverlay(overlay)
}
func (p *LogPane) PopModalOverlay() {
	p.inputProcessor.PopModalOverlay()
}
func (p *LogPane) PopModalOverlays(index int) {
	p.inputProcessor.PopModalOverlays(index)
}

func (p *LogPane) GetHelp() input.Help { return p.inputProcessor.GetHelp() }

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
		renderer:    renderer,
		dimensions:  dimensions,
		stylesheet:  stylesheet,
		condition:   condition,
		titleString: titleString,
		logReader:   logReader,
	}
}
