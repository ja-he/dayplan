package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
)

// HelpPane conditionally be hidden or display a set of keyboad shortcuts.
type HelpPane struct {
	renderer   ui.ConstrainedRenderer
	dimensions func() (x, y, w, h int)
	stylesheet styling.Stylesheet
	condition  func() bool

	inputProcessor input.ModalInputProcessor
	parent         ui.FocusQueriable
}

// EnsureHidden informs the pane that it is not being shown so that it can take
// potential actions to ensure that, e.g., hide the terminal cursor, if
// necessary.
func (p *HelpPane) EnsureHidden() {}

// Condition returns whether this pane should be visible.
func (p *HelpPane) Condition() bool { return p.condition() }

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *HelpPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *HelpPane) GetPositionInfo(x, y int) ui.PositionInfo { return nil }

func (p *HelpPane) CapturesInput() bool {
	return p.inputProcessor != nil && p.inputProcessor.CapturesInput()
}
func (p *HelpPane) ProcessInput(key input.Key) bool {
	return p.inputProcessor != nil && p.inputProcessor.ProcessInput(key)
}

func (p *HelpPane) HasFocus() bool                     { return p.parent.HasFocus() && p.parent.Focusses() == p }
func (p *HelpPane) Focusses() ui.FocussablePane        { return nil }
func (p *HelpPane) SetParent(parent ui.FocusQueriable) { p.parent = parent }

func (p *HelpPane) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index int) {
	return p.inputProcessor.ApplyModalOverlay(overlay)
}
func (p *HelpPane) PopModalOverlay()           { p.inputProcessor.PopModalOverlay() }
func (p *HelpPane) PopModalOverlays(index int) { p.inputProcessor.PopModalOverlays(index) }

// Draw draws the help popup.
func (p *HelpPane) Draw() {
	if p.condition() {

		x, y, w, h := p.Dimensions()
		p.renderer.DrawBox(x, y, w, h, p.stylesheet.Help)

		keysDrawn := 0
		const border = 1
		const maxKeyWidth = 20
		const pad = 1
		keyOffset := x + border
		descriptionOffset := keyOffset + maxKeyWidth + pad

		drawMapping := func(keys, description string) {
			p.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keys)), y+border+keysDrawn, len([]rune(keys)), 1, p.stylesheet.Help.DefaultEmphasized().Bolded(), keys)
			p.renderer.DrawText(descriptionOffset, y+border+keysDrawn, w, h, p.stylesheet.Help.Italicized(), description)
			keysDrawn++
		}

		drawOpposedMapping := func(keyA, keyB, description string) {
			sepText := "/"
			p.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keyB))-len(sepText)-len([]rune(keyA)), y+border+keysDrawn, len([]rune(keyA)), 1, p.stylesheet.Help.DefaultEmphasized().Bolded(), keyA)
			p.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keyB))-len(sepText), y+border+keysDrawn, len(sepText), 1, p.stylesheet.Help, sepText)
			p.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keyB)), y+border+keysDrawn, len([]rune(keyB)), 1, p.stylesheet.Help.DefaultEmphasized().Bolded(), keyB)
			p.renderer.DrawText(descriptionOffset, y+border+keysDrawn, w, h, p.stylesheet.Help.Italicized(), description)
			keysDrawn++
		}

		space := func() { drawMapping("", "") }

		drawMapping("?", "toggle help")
		space()

		drawMapping("<lmb>[+<move down>]", "create or edit event")
		drawMapping("<rmb>", "split event (in event view)")
		drawMapping("<mmb>", "delete event")
		drawMapping("<ctrl-lmb>+<move>", "move event with following")
		space()

		drawOpposedMapping("<c-u>", "<c-d>", "scroll up / down")
		drawOpposedMapping("k", "j", "scroll up / down")
		drawOpposedMapping("g", "G", "scroll to top / bottom")
		space()

		drawOpposedMapping("+", "-", "zoom in / out")
		space()

		drawOpposedMapping("h", "l", "go to previous / next day")
		space()

		drawOpposedMapping("i", "<esc>", "narrow / broaden view")
		space()

		drawMapping("w", "write day to file")
		drawMapping("c", "clear day (remove all events)")
		drawMapping("q", "quit (unwritten data is lost)")
		space()

		drawMapping("S", "toggle summary")
		drawMapping("E", "toggle debug log")
		space()

		drawMapping("u", "update weather (requires some envvars)")
		space()
	}
}

// NewHelpPane constructs and returns a new HelpPane.
func NewHelpPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	condition func() bool,
) *HelpPane {
	return &HelpPane{
		renderer:   renderer,
		dimensions: dimensions,
		stylesheet: stylesheet,
		condition:  condition,
	}
}
