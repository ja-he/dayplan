// Package edit implements generic interfaces for editing of objects (by the
// user).
package edit

import (
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
)

// Editor is an interface for editing of objects (by the user).
type Editor interface {
	GetName() string

	// Write the state of the editor.
	Write()

	// Quit the editor.
	Quit()

	// AddQuitCallback adds a callback that is called when the editor is quit.
	AddQuitCallback(func())

	// GetFieldCount returns the number of fields of the editor.
	GetFieldCount() int

	// GetPane returns a pane that represents this editor.
	GetPane(
		renderer ui.ConstrainedRenderer,
		visible func() bool,
		inputConfig input.InputConfig,
		stylesheet styling.Stylesheet,
		cursorController ui.TextCursorController,
	) (ui.Pane, error)
}
