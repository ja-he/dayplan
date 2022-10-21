package panes

import (
	"fmt"
	"math"

	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
)

// An EventsPane displays a single days events.
// It can be configured to display events with more decorations and padding
// (e.g., when displaying a single day in the UI), or to be space efficient and
// hide some details (e.g., for showing events as part of multiple EventPanes in
// in the month view.
type EventsPane struct {
	Leaf

	day func() *model.Day

	styleForCategory func(model.Category) (styling.DrawStyling, error)

	viewParams *ui.ViewParams
	cursor     *ui.MouseCursorPos

	pad             int
	drawTimestamps  bool
	drawNames       bool
	drawCat         bool
	isCurrentDay    func() bool
	getCurrentEvent func() *model.Event
	mouseMode       func() bool

	// TODO: get rid of this
	positions map[*model.Event]util.Rect
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *EventsPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// GetPositionInfo returns information on a requested position in this pane.
// Importantly, when there is an event at the position, it will inform of that
// in detail.
func (p *EventsPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return p.getEventForPos(x, y)
}

// Draw draws this pane.
func (p *EventsPane) Draw() {
	x, y, w, h := p.Dimensions()
	style := p.stylesheet.Normal
	if p.HasFocus() {
		style = p.stylesheet.NormalEmphasized
	}
	p.renderer.DrawBox(x, y, w, h, style)

	day := p.day()

	if day == nil {
		log.Debug().Msg("current day nil on render; skipping")
		// TODO: just draw this, man
		return
	}
	p.positions = p.computeRects(day, x+p.pad, y, w-(2*p.pad), h)
	for _, e := range day.Events {
		style, err := p.styleForCategory(e.Cat)
		if err != nil {
			log.Error().Err(err).Str("category-name", e.Cat.Name).Msg("an error occurred getting category style")
			style = p.stylesheet.CategoryFallback
		}
		if !p.isCurrentDay() {
			style = style.DefaultDimmed()
		}

		// based on event state, draw a box or maybe a smaller one, or ...
		pos := p.positions[e]
		var timestampWidth int
		if p.drawTimestamps {
			timestampWidth = 5
		} else {
			timestampWidth = 0
		}
		var hovered *ui.EventsPanePositionInfo
		if p.mouseMode() {
			hovered = p.getEventForPos(p.cursor.X, p.cursor.Y)
		}

		if p.getCurrentEvent() == e {
			style = style.Invert()
		}

		var bodyStyling styling.DrawStyling = style
		var bottomStyling styling.DrawStyling = style
		var nameStyling styling.DrawStyling = style

		namePadding := 1
		nameWidth := pos.W - (2 * namePadding) - timestampWidth

		if p.mouseMode() && hovered != nil && hovered.Event == e && hovered.EventBoxPart != ui.EventBoxNowhere {
			selectionStyling := style.DefaultEmphasized()

			switch hovered.EventBoxPart {

			case ui.EventBoxBottomRight:
				bottomStyling = selectionStyling.Bolded()

			case ui.EventBoxInterior:
				bottomStyling = selectionStyling
				bodyStyling = selectionStyling
				nameStyling = selectionStyling

			case ui.EventBoxTopEdge:
				nameStyling = selectionStyling.Bolded()

			default:
				panic(fmt.Sprint("don't know this hover state:", hovered.EventBoxPart.ToString()))
			}
		}

		var topTimestampStyling = bodyStyling.NormalizeFromBG(0.4)
		var botTimestampStyling = bottomStyling.NormalizeFromBG(0.4)

		p.renderer.DrawBox(pos.X, pos.Y, pos.W, pos.H, bodyStyling)

		if p.drawTimestamps {
			p.renderer.DrawText(pos.X+pos.W-5, pos.Y, 5, 1, topTimestampStyling, e.Start.ToString())
		}

		p.renderer.DrawBox(pos.X, pos.Y+pos.H-1, pos.W, 1, bottomStyling)
		if p.drawTimestamps {
			p.renderer.DrawText(pos.X+pos.W-5, pos.Y+pos.H-1, 5, 1, botTimestampStyling, e.End.ToString())
		}

		if p.drawNames {
			p.renderer.DrawText(pos.X+1, pos.Y, nameWidth, 1, nameStyling, util.TruncateAt(e.Name, nameWidth))
		}
		if p.drawCat && pos.H > 1 {
			var catStyling = bodyStyling.NormalizeFromBG(0.2).Unbolded().Italicized()
			if pos.H == 2 {
				catStyling = bottomStyling.NormalizeFromBG(0.2).Unbolded().Italicized()
			}
			catWidth := pos.W - 2 - 1
			p.renderer.DrawText(pos.X+pos.W-1-catWidth, pos.Y+1, catWidth, 1, catStyling, util.TruncateAt(e.Cat.Name, catWidth))
		}

	}
}

func (p *EventsPane) getEventForPos(x, y int) *ui.EventsPanePositionInfo {
	dimX, _, dimW, _ := p.Dimensions()

	if x >= dimX &&
		x < (dimX+dimW) {
		currentDay := p.day()
		for i := len(currentDay.Events) - 1; i >= 0; i-- {
			eventPos := p.positions[currentDay.Events[i]]
			if eventPos.Contains(x, y) {
				var hover ui.EventBoxPart
				switch {
				case y == (eventPos.Y+eventPos.H-1) && x >= eventPos.X+eventPos.W-5:
					hover = ui.EventBoxBottomRight
				case y == (eventPos.Y):
					hover = ui.EventBoxTopEdge
				default:
					hover = ui.EventBoxInterior
				}
				return &ui.EventsPanePositionInfo{
					Event:        currentDay.Events[i],
					EventBoxPart: hover,
					Time:         p.viewParams.TimeAtY(y),
				}
			}
		}
	}
	return &ui.EventsPanePositionInfo{
		Event:        nil,
		EventBoxPart: ui.EventBoxNowhere,
		Time:         p.viewParams.TimeAtY(y),
	}
}

func (p *EventsPane) computeRects(day *model.Day, offsetX, offsetY, width, height int) map[*model.Event]util.Rect {
	activeStack := make([]*model.Event, 0)
	positions := make(map[*model.Event]util.Rect)
	for _, e := range day.Events {
		// remove all stacked elements that have finished
		for i := len(activeStack) - 1; i >= 0; i-- {
			if e.Start.IsAfter(activeStack[i].End) || e.Start == activeStack[i].End {
				activeStack = activeStack[:i]
			} else {
				break
			}
		}
		activeStack = append(activeStack, e)
		// based on event state, draw a box or maybe a smaller one, or ...
		x := offsetX
		y := p.viewParams.YForTime(e.Start) + offsetY
		w := width
		h := p.viewParams.YForTime(e.End) + offsetY - y

		// scale the width by 3/4 for every extra item on the stack, so for one
		// item stacked underneath the current items width will be (3/4) ** 1 = 75%
		// of the original width, for four it would be (3/4) ** 4 = (3**4)/(4**4)
		// or 31.5 % of the width, etc.
		widthFactor := 0.75
		w = int(float64(w) * math.Pow(widthFactor, float64(len(activeStack)-1)))
		x += (width - w)

		// make the current event wider by 1 on either side
		if e == p.getCurrentEvent() {
			x -= 1
			w += 2
		}

		positions[e] = util.Rect{X: x, Y: y, W: w, H: h}
	}
	return positions
}

// NewEventsPane constructs and returns a new EventsPane.
func NewEventsPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	inputProcessor input.ModalInputProcessor,
	day func() *model.Day,
	styleForCategory func(model.Category) (styling.DrawStyling, error),
	viewParams *ui.ViewParams,
	cursor *ui.MouseCursorPos,
	pad int,
	drawTimestamps bool,
	drawNames bool,
	drawCat bool,
	isCurrentDay func() bool,
	getCurrentEvent func() *model.Event,
	mouseMode func() bool,
) *EventsPane {
	return &EventsPane{
		Leaf: Leaf{
			Base: Base{
				ID:             ui.GeneratePaneID(),
				InputProcessor: inputProcessor,
			},
			renderer:   renderer,
			dimensions: dimensions,
			stylesheet: stylesheet,
		},
		day:              day,
		styleForCategory: styleForCategory,
		viewParams:       viewParams,
		cursor:           cursor,
		pad:              pad,
		drawTimestamps:   drawTimestamps,
		drawNames:        drawNames,
		drawCat:          drawCat,
		isCurrentDay:     isCurrentDay,
		getCurrentEvent:  getCurrentEvent,
		mouseMode:        mouseMode,
		positions:        make(map[*model.Event]util.Rect, 0),
	}
}
