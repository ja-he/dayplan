// TODO: this should be moved to 'control/editor' imo
package input

// TextEditMode enumerates the possible modes of modal text editing.
type TextEditMode = int

const (
	// TextEditModeNormal is the "normal mode" of the editor, i.E. key inputs are
	// not used for input directly, but for navigation and manipulation of the
	// text.
	TextEditModeNormal = 0
	// TextEditModeInsert is the "insert mode" of the editor, i.E. key inputs are
	// used for input directly.
	TextEditModeInsert = 1
)
