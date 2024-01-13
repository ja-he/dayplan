// Package edit implements generic interfaces for editing of objects (by the
// user).
package edit

// Editor is an interface for editing of objects (by the user).
type Editor interface {
	IsActiveAndFocussed() (bool, bool)

	GetName() string

	GetType() string

	// Write the state of the editor.
	Write()

	// Quit the editor.
	Quit()

	// AddQuitCallback adds a callback that is called when the editor is quit.
	AddQuitCallback(func())
}
