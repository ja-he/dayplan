// Package edit implements generic interfaces for editing of objects (by the
// user).
package edit

// Editor is an interface for editing of objects (by the user).
type Editor interface {

	// GetStatus informs on whether the editor is active and
	// selected and focussed.
	GetStatus() EditorStatus

	GetName() string

	GetType() string

	// Write the state of the editor.
	Write()

	// Quit the editor.
	Quit()

	// AddQuitCallback adds a callback that is called when the editor is quit.
	AddQuitCallback(func())
}

// EditorStatus informs on the status of an editor with respect to its
// selection.
type EditorStatus string

const (
	// EditorInactive indicates that the editor is not active.
	//
	// In other words, the editor is not in the active chain.
	EditorInactive EditorStatus = "inactive"

	// EditorSelected indicates that the editor is the one currently selected for
	// editing within its parent, while its parent is focussed.
	//
	// In other words, the editor is not yet in the active chain but just beyond
	// the end of it, and currently the "closest" to being added to the end of
	// the chain, as only some sort of "confirm-selection" operation in the
	// parent is now needed to focus this editor.
	EditorSelected EditorStatus = "selected"

	// EditorFocussed indicates the editor is the editor that currently has focus
	// and receives inputs.
	//
	// In other words, the editor is on the active chain and is the lowestmost on
	// the chain, ie. the end of the chain.
	EditorFocussed EditorStatus = "focussed"

	// EditorDescendantActive indicates that an descendent of the editor (a child,
	// grandchild, ...) is active.
	//
	// In other words, the editor is in the active chain but is not the end of
	// the chain, ie. there is at least one lower editor on the chain (a
	// descendant). The editor may be the beginning of the active chain.
	EditorDescendantActive EditorStatus = "descendant-active"
)
