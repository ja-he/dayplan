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
	Leaf

	titleString func() string
	days        func() []*model.Day

	categories *styling.CategoryStyling
}

// EnsureHidden informs the pane that it is not being shown so that it can take
// potential actions to ensure that, e.g., hide the terminal cursor, if
// necessary.
func (p *SummaryPane) EnsureHidden() {}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *SummaryPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw draws the time summary view over top of all previously drawn contents,
// if it is currently active.
func (p *SummaryPane) Draw() {

	if p.IsVisible() {
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
		Leaf: Leaf{
			Base: Base{
				ID:             ui.GeneratePaneID(),
				InputProcessor: inputProcessor,
				Visible:        condition,
			},
			renderer:   renderer,
			dimensions: dimensions,
			stylesheet: stylesheet,
		},
		titleString: titleString,
		days:        days,
		categories:  categories,
	}
}
