package panes

import (
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/ui"
)

// Base is the base data necessary for a UI pane and provides a base
// implementation using them.
//
// Note that constructing this value that you need to assign the ID.
type Base struct {
	ID             ui.PaneID
	Parent         ui.PaneQuerier
	InputProcessor input.ModalInputProcessor
	Visible        func() bool
}

// Identify returns the panes ID.
func (p *Base) Identify() ui.PaneID {
	if p.ID == ui.NonePaneID {
		// NOTE(ja-he): generally, the none-value is OK; put this here to catch errors early
		panic("pane has not been assigned an ID")
	}
	return p.ID
}

// SetParent sets the pane's parent.
func (p *Base) SetParent(parent ui.PaneQuerier) { p.Parent = parent }

// IsVisible indicates whether the pane is visible.
func (p *Base) IsVisible() bool { return p.Visible == nil || p.Visible() }
