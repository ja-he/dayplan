package tui

import (
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

type SummaryPane struct {
	renderer   ConstrainedRenderer
	dimensions func() (x, y, w, h int)

	condition   func() bool
	titleString func() string
	days        func() []*model.Day

	categories *category_style.CategoryStyling
}

func (p *SummaryPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draws the time summary view over top of all previously drawn contents, if it
// is currently active.
func (p *SummaryPane) Draw() {
	style := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	if p.condition() {
		x, y, w, h := p.Dimensions()

		p.renderer.DrawBox(x, y, w, h, style)
		title := p.titleString()
		p.renderer.DrawBox(x, y, w, 1, style.Background(tcell.ColorLightGrey))
		p.renderer.DrawText(x+(w/2-len(title)/2), y, len(title), 1, style.Background(tcell.ColorLightGrey).Bold(true), title)

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
			style, _ := p.categories.GetStyle(category)
			catLen := 20
			durationLen := 20
			barWidth := int(float64(duration) / float64(maxDuration) * float64(w-catLen-durationLen))
			p.renderer.DrawBox(x+catLen+durationLen, y+row, barWidth, 1, style)
			p.renderer.DrawText(x, y+row, catLen, 1, tcell.StyleDefault, util.TruncateAt(category.Name, catLen))
			p.renderer.DrawText(x+catLen, y+row, durationLen, 1, style, "("+util.DurationToString(duration)+")")
			row++
		}
	}
}

func (p *SummaryPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}
