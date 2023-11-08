package ui

import (
	"github.com/ja-he/dayplan/internal/input"
)

// BasePane is the base data necessary for a UI pane and provides a base
// implementation using them.
//
// Note that constructing this value that you need to assign the ID.
type BasePane struct {
	ID             PaneID
	Parent         PaneQuerier
	InputProcessor input.ModalInputProcessor
	Visible        func() bool
}

// Identify returns the panes ID.
func (p *BasePane) Identify() PaneID {
	if p.ID == NonePaneID {
		// NOTE(ja-he): generally, the none-value is OK; put this here to catch errors early
		panic("pane has not been assigned an ID")
	}
	return p.ID
}

// SetParent sets the pane's parent.
func (p *BasePane) SetParent(parent PaneQuerier) { p.Parent = parent }

// IsVisible indicates whether the pane is visible.
func (p *BasePane) IsVisible() bool { return p.Visible == nil || p.Visible() }
