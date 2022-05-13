package processors

import (
	"github.com/gdamore/tcell/v2"

	"github.com/ja-he/dayplan/src/control/action"
	"github.com/ja-he/dayplan/src/input"
)

type TextInputProcessor struct {
	mappings map[input.Key]action.Action

	runeCallback func(r rune)
}

func (p *TextInputProcessor) ProcessInput(key input.Key) bool {
	if key.Key == tcell.KeyRune {
		p.runeCallback(key.Ch)
	} else {
		action, mappingExists := p.mappings[key]
		if mappingExists {
			action.Do()
		}
	}

	// I think we will always want a text processor to capture all input, right?
	// If I pressed <c-d> and we didn't have a specific mapping from the input
	// processor, we wouldn't want the root pane to end up handling the input.
	return true
}

func (p *TextInputProcessor) CapturesInput() bool {
	// I think we will always want a text processor to take this precedence.
	return true
}

func (p *TextInputProcessor) GetHelp() input.Help { return nil }

func NewTextInputProcessor(
	normalModeMappings map[input.Key]action.Action,
	runeCallback func(r rune),
) *TextInputProcessor {
	return &TextInputProcessor{
		mappings:     normalModeMappings,
		runeCallback: runeCallback,
	}
}
