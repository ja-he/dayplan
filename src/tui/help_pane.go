package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/colors"
	"github.com/ja-he/dayplan/src/ui"
)

type HelpPane struct {
	renderer   ConstrainedRenderer
	dimensions func() (x, y, w, h int)
	condition  func() bool
}

func (p *HelpPane) Condition() bool { return p.condition() }

func (p *HelpPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *HelpPane) GetPositionInfo(x, y int) ui.PositionInfo { return nil }

// Draw the help popup.
func (p *HelpPane) Draw() {
	if p.condition() {

		helpStyle := tcell.StyleDefault.Background(tcell.ColorLightGrey)
		keyStyle := colors.DefaultEmphasize(helpStyle).Bold(true)
		descriptionStyle := helpStyle.Italic(true)

		x, y, w, h := p.Dimensions()
		p.renderer.DrawBox(x, y, w, h, helpStyle)

		keysDrawn := 0
		const border = 1
		const maxKeyWidth = 20
		const pad = 1
		keyOffset := x + border
		descriptionOffset := keyOffset + maxKeyWidth + pad

		drawMapping := func(keys, description string) {
			p.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keys)), y+border+keysDrawn, len([]rune(keys)), 1, keyStyle, keys)
			p.renderer.DrawText(descriptionOffset, y+border+keysDrawn, w, h, descriptionStyle, description)
			keysDrawn++
		}

		drawOpposedMapping := func(keyA, keyB, description string) {
			sepText := "/"
			p.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keyB))-len(sepText)-len([]rune(keyA)), y+border+keysDrawn, len([]rune(keyA)), 1, keyStyle, keyA)
			p.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keyB))-len(sepText), y+border+keysDrawn, len(sepText), 1, helpStyle, sepText)
			p.renderer.DrawText(keyOffset+maxKeyWidth-len([]rune(keyB)), y+border+keysDrawn, len([]rune(keyB)), 1, keyStyle, keyB)
			p.renderer.DrawText(descriptionOffset, y+border+keysDrawn, w, h, descriptionStyle, description)
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
