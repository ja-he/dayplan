package processors

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/ja-he/dayplan/internal/control/action"
	"github.com/ja-he/dayplan/internal/input"
)

// TextInputProcessor is a SimpleInputProcessor specifically for text input.
// It can have a number of defined mappings for non-runes (e.g. ESC for a
// callback to remove this processor as an overlay).
// Any runes it is asked to process will be given to its callback function for
// runes, which could, e.g., append the given rune to a string.
type TextInputProcessor struct {
	mappings map[input.Key]action.Action

	runeCallback func(r rune)
}

// ProcessInput attempts to process the provided input.
// Returns whether the provided input "applied", i.E. the processor performed
// an action based on the input.
func (p *TextInputProcessor) ProcessInput(key input.Key) bool {
	if key.Key == tcell.KeyRune {
		p.runeCallback(key.Ch)
		return true
	} else {
		action, mappingExists := p.mappings[key]
		if mappingExists {
			action.Do()
			return true
		} else {
			return false
		}
	}
}

// CapturesInput returns whether this processor "captures" input, i.E. whether
// it ought to take priority in processing over other processors.
// This is useful, e.g., for prioritizing processors whith partial input
// sequences or for such overlays, that are to take complete priority by
// completely gobbling all input.
func (p *TextInputProcessor) CapturesInput() bool {
	// I think we will always want a text processor to take this precedence.
	return true
}

// GetHelp returns the input help map for this processor.
func (p *TextInputProcessor) GetHelp() input.Help {
	result := input.Help{}
	for k, a := range p.mappings {
		result[input.ToConfigIdentifierString(k)] = a.Explain()
	}
	return result
}

// NewTextInputProcessor returns a pointer to a new NewTextInputProcessor.
func NewTextInputProcessor(
	normalModeMappings map[input.Keyspec]action.Action,
	runeCallback func(r rune),
) (*TextInputProcessor, error) {
	mappings := map[input.Key]action.Action{}
	for keyspec, action := range normalModeMappings {
		keys, err := input.ConfigKeyspecToKeys(keyspec)
		if err != nil {
			return nil, fmt.Errorf("could not convert '%s' to keys (%s)", keyspec, err.Error())
		}
		if len(keys) != 1 {
			return nil, fmt.Errorf("keyspec '%s' for text processor has not exactly one key (but %d)", keyspec, len(keys))
		}
		mappings[keys[0]] = action
	}
	return &TextInputProcessor{
		mappings:     mappings,
		runeCallback: runeCallback,
	}, nil
}
