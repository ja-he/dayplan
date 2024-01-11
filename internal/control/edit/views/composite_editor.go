package views

// StringEditorView allows inspection of a string editor.
type CompositeEditorView interface {

	// IsActive signals whether THIS is active. (SHOULD BE MOVED TO A MORE GENERIC INTERFACE)
	IsActiveAndFocussed() (bool, bool)

	GetName() string

	// TODO: more
}
