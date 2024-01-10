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
	IsActiveAndFocussed() (bool, bool)

	GetName() string

	GetType() string
	GetSummary() SummaryEntry

	// Write the state of the editor.
	Write()

	// Quit the editor.
	Quit()

	// AddQuitCallback adds a callback that is called when the editor is quit.
	AddQuitCallback(func())

	// GetPane returns a pane that represents this editor.
	GetPane(
		renderer ui.ConstrainedRenderer,
		visible func() bool,
		inputConfig input.InputConfig,
		stylesheet styling.Stylesheet,
		cursorController ui.CursorLocationRequestHandler,
	) (ui.Pane, error)
}

type SummaryEntry struct {
	Representation any
	Represents     Editor
}
