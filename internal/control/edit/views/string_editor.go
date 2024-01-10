package views

import "github.com/ja-he/dayplan/internal/input"

// StringEditorView allows inspection of a string editor.
type StringEditorView interface {

	// IsActive signals whether THIS is active. (SHOULD BE MOVED TO A MORE GENERIC INTERFACE)
	IsActiveAndFocussed() (bool, bool)

	// GetMode returns the current mode of the editor.
	GetMode() input.TextEditMode

	// GetCursorPos returns the current cursor position in the string, 0 being
	// the first character.
	GetCursorPos() int

	// GetContent returns the current (edited) contents.
	GetContent() string

	GetName() string

	// TODO: more
}
