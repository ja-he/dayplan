package panes

import (
	"fmt"
	"math"

	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
	"github.com/lucasb-eyer/go-colorful"
)

// PerfPane is an ephemeral pane used for showing debug info during normal
// usage.
type PerfPane struct {
	renderer ui.ConstrainedRenderer

	dimensions func() (x, y, w, h int)
	condition  func() bool

	renderTime          util.MetricsGetter
	eventProcessingTime util.MetricsGetter
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *PerfPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw draws this pane.
func (p *PerfPane) Draw() {
	if !p.condition() {
		return
	}

	renderAvg := p.renderTime.Avg()
	renderLast := p.renderTime.GetLast()
	eventAvg := p.eventProcessingTime.Avg()
	eventLast := p.eventProcessingTime.GetLast()

	x, y, w, h := p.dimensions()
	lastWidth := len(" render time: ....... xs ")
	avgWidth := w - lastWidth

	defaultStyle := styling.StyleFromHex("#000000", "#f0f0f0")
	bad := colorful.Color{R: 1.0, G: 0.8, B: 0.8}
	hue, _, ltn := bad.Hsl()

	renderSat := float64(0)
	if renderLast > renderAvg {
		renderSat = math.Min(float64(renderLast-renderAvg)/float64(renderAvg), 1.0)
	}
	renderStyle := styling.StyleFromColors(
		colorful.Hsl(0, 0, 0), // black
		colorful.Hsl(hue, renderSat, ltn),
	)

	eventSat := float64(0)
	if eventLast > eventAvg {
		eventSat = math.Min(float64(eventLast-eventAvg)/float64(eventAvg), 1.0)
	}
	eventStyle := styling.StyleFromColors(
		colorful.Hsl(0, 0, 0), // black
		colorful.Hsl(hue, eventSat, ltn),
	)

	p.renderer.DrawBox(x, y, w, h, defaultStyle)

	p.renderer.DrawText(x, y, lastWidth, 1, renderStyle, fmt.Sprintf(" render time: % 7d µs ", renderLast))
	p.renderer.DrawText(x, y+1, lastWidth, 1, eventStyle, fmt.Sprintf(" input  time: % 7d µs ", eventLast))

	p.renderer.DrawText(x+lastWidth, y, avgWidth, 1, defaultStyle, fmt.Sprintf(" render avg ~ % 7d µs", renderAvg))
	p.renderer.DrawText(x+lastWidth, y+1, avgWidth, 1, defaultStyle, fmt.Sprintf(" input  avg ~ % 7d µs", eventAvg))
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *PerfPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}

// Condition indicates whether this pane is active.
func (p *PerfPane) Condition() bool {
	return p.condition()
}

// EnsureHidden informs the pane that it is not being shown so that it can take
// potential actions to ensure that, e.g., hide the terminal cursor, if
// necessary.
func (p *PerfPane) EnsureHidden() {}

// NewPerfPane constructs and returns a new PerfPane.
func NewPerfPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	condition func() bool,
	renderTime util.MetricsGetter,
	eventProcessingTime util.MetricsGetter,
) *PerfPane {
	return &PerfPane{
		renderer:            renderer,
		dimensions:          dimensions,
		condition:           condition,
		renderTime:          renderTime,
		eventProcessingTime: eventProcessingTime,
	}
}
