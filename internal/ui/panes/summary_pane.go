package panes

import (
	"sort"
	"time"

	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SummaryPane shows a summary of the set of days it is provided.
// It shows all events' times summed up (by Summarize, meaning without counting
// any time multiple times) and visualizes the results in simple bars.
type SummaryPane struct {
	ui.LeafPane

	titleString func() string
	summary     func() (map[model.CategoryName]time.Duration, error)

	categories    storage.CategoryProvider
	categoryStyle func(model.CategoryName) (styling.DrawStyling, error)

	log zerolog.Logger
}

// EnsureHidden informs the pane that it is not being shown so that it can take
// potential actions to ensure that, e.g., hide the terminal cursor, if
// necessary.
func (p *SummaryPane) EnsureHidden() {}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *SummaryPane) Dimensions() (x, y, w, h int) {
	return p.Dims()
}

// Draw draws the time summary view over top of all previously drawn contents,
// if it is currently active.
func (p *SummaryPane) Draw() {

	if p.IsVisible() {
		x, y, w, h := p.Dimensions()

		p.Renderer.DrawBox(x, y, w, h, p.Stylesheet.SummaryDefault)
		title := p.titleString()
		p.Renderer.DrawBox(x, y, w, 1, p.Stylesheet.SummaryTitleBox)
		p.Renderer.DrawText(x+(w/2-len(title)/2), y, len(title), 1, p.Stylesheet.SummaryTitleBox, title)

		summary, err := p.summary()
		if err != nil {
			msg1 := "ERROR"
			msg2 := "(see log)"
			p.Renderer.DrawText(x+(w/2-len(msg1)/2), y+(h/2-1), len(msg1), 1,
				p.Stylesheet.CategoryFallback.Bolded(),
				msg1)
			p.Renderer.DrawText(x+(w/2-len(msg2)/2), y+(h/2), len(msg2), 1,
				p.Stylesheet.CategoryFallback.Italicized(),
				msg2)
			p.log.Error().Err(err).Msg("could not get summary")
			return
		}
		maxDuration := time.Duration(0)
		categories := make([]*model.Category, len(summary))
		{ // get sorted keys to have deterministic order
			i := 0
			for categoryName, duration := range summary {
				c := p.categories.GetCategory(categoryName)
				if c == nil {
					p.log.Error().Str("category", string(categoryName)).Msg("nil category (but ok)")
					return
				}
				categories[i] = c
				if duration > maxDuration {
					maxDuration = duration
				}
				i++
			}
			sort.Sort(model.ByName(categories))
		}
		row := 2
		for _, category := range categories {
			duration := summary[category.Name]

			var err error
			var categoryStyling styling.DrawStyling
			categoryStyling, err = p.categoryStyle(category.Name)
			if err != nil {
				p.log.Warn().Msgf("Failed to get style for cat '%s', using fallback.", category.Name)
				categoryStyling = p.Stylesheet.CategoryFallback
			}
			catLen := 20
			durationLen := 20
			barWidth := int(float64(duration) / float64(maxDuration) * float64(w-catLen-durationLen))
			p.Renderer.DrawBox(x+catLen+durationLen, y+row, barWidth, 1, categoryStyling)
			p.Renderer.DrawText(x, y+row, catLen, 1, p.Stylesheet.SummaryDefault, util.TruncateAt(string(category.Name), catLen))
			p.Renderer.DrawText(x+catLen, y+row, durationLen, 1, categoryStyling, "("+duration.String()+")")
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
	summary func() (map[model.CategoryName]time.Duration, error),
	categories storage.CategoryProvider,
	getCategoryStyle func(model.CategoryName) (styling.DrawStyling, error),
	inputProcessor input.ModalInputProcessor,
) *SummaryPane {
	return &SummaryPane{
		LeafPane: ui.LeafPane{
			BasePane: ui.BasePane{
				ID:             ui.GeneratePaneID(),
				InputProcessor: inputProcessor,
				Visible:        condition,
			},
			Renderer:   renderer,
			Dims:       dimensions,
			Stylesheet: stylesheet,
		},
		titleString:   titleString,
		summary:       summary,
		categories:    categories,
		categoryStyle: getCategoryStyle,
		log:           log.With().Str("component", "summary-pane").Logger(),
	}
}
