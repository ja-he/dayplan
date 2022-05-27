package panes

import (
	"sort"

	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

// SummaryPane shows a summary of the set of days it is provided.
// It shows all events' times summed up (by Summarize, meaning without counting
// any time multiple times) and visualizes the results in simple bars.
type SummaryPane struct {
	renderer   ui.ConstrainedRenderer
	dimensions func() (x, y, w, h int)
	stylesheet styling.Stylesheet

	condition   func() bool
	titleString func() string
	days        func() []*model.Day

	Parent         ui.FocusQueriable
	inputProcessor input.ModalInputProcessor

	categories *styling.CategoryStyling
}

// EnsureHidden informs the pane that it is not being shown so that it can take
// potential actions to ensure that, e.g., hide the terminal cursor, if
// necessary.
func (p *SummaryPane) EnsureHidden() {}

// Condition returns whether this pane should be visible.
func (p *SummaryPane) Condition() bool { return p.condition() }

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *SummaryPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw draws the time summary view over top of all previously drawn contents,
// if it is currently active.
func (p *SummaryPane) Draw() {

	if p.condition() {
		x, y, w, h := p.Dimensions()

		p.renderer.DrawBox(x, y, w, h, p.stylesheet.SummaryDefault)
		title := p.titleString()
		p.renderer.DrawBox(x, y, w, 1, p.stylesheet.SummaryTitleBox)
		p.renderer.DrawText(x+(w/2-len(title)/2), y, len(title), 1, p.stylesheet.SummaryTitleBox, title)

		summary := make(map[model.Category]int)

		days := p.days()
		for i := range days {
			if days[i] == nil {
				return
			}
			tmpSummary := days[i].SumUpByCategory()
			for k, v := range tmpSummary {
				summary[k] += v
			}
		}

		maxDuration := 0
		categories := make([]model.Category, len(summary))
		{ // get sorted keys to have deterministic order
			i := 0
			for category, duration := range summary {
				categories[i] = category
				if duration > maxDuration {
					maxDuration = duration
				}
				i++
			}
			sort.Sort(model.ByName(categories))
		}
		row := 2
		for _, category := range categories {
			duration := summary[category]
			style, err := p.categories.GetStyle(category)
			if err != nil {
				style = p.stylesheet.CategoryFallback
			}
			categoryStyling := style
			catLen := 20
			durationLen := 20
			barWidth := int(float64(duration) / float64(maxDuration) * float64(w-catLen-durationLen))
			p.renderer.DrawBox(x+catLen+durationLen, y+row, barWidth, 1, categoryStyling)
			p.renderer.DrawText(x, y+row, catLen, 1, p.stylesheet.SummaryDefault, util.TruncateAt(category.Name, catLen))
			p.renderer.DrawText(x+catLen, y+row, durationLen, 1, categoryStyling, "("+util.DurationToString(duration)+")")
			row++
		}
	}
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *SummaryPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}

func (p *SummaryPane) CapturesInput() bool {
	return p.inputProcessor != nil && p.inputProcessor.CapturesInput()
}
func (p *SummaryPane) ProcessInput(key input.Key) bool {
	return p.inputProcessor != nil && p.inputProcessor.ProcessInput(key)
}

func (p *SummaryPane) HasFocus() bool              { return p.Parent.HasFocus() && p.Parent.Focusses() == p }
func (p *SummaryPane) Focusses() ui.FocussablePane { return nil }

func (p *SummaryPane) SetParent(parent ui.FocusQueriable) {
	p.Parent = parent
}

func (p *SummaryPane) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index uint) {
	return p.inputProcessor.ApplyModalOverlay(overlay)
}
func (p *SummaryPane) PopModalOverlay() error {
	return p.inputProcessor.PopModalOverlay()
}
func (p *SummaryPane) PopModalOverlays(index uint) {
	p.inputProcessor.PopModalOverlays(index)
}

func (p *SummaryPane) GetHelp() input.Help { return p.inputProcessor.GetHelp() }

// NewSummaryPane constructs and returns a new SummaryPane.
func NewSummaryPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	condition func() bool,
	titleString func() string,
	days func() []*model.Day,
	categories *styling.CategoryStyling,
	inputProcessor input.ModalInputProcessor,
) *SummaryPane {
	return &SummaryPane{
		renderer:       renderer,
		dimensions:     dimensions,
		stylesheet:     stylesheet,
		condition:      condition,
		titleString:    titleString,
		days:           days,
		categories:     categories,
		inputProcessor: inputProcessor,
	}
}
