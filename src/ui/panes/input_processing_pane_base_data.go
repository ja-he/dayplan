package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/ui"
)

type InputProcessingPaneBaseData struct {
	ID             ui.PaneID
	Parent         ui.FocusQueriable
	InputProcessor input.ModalInputProcessor
	Visible        func() bool
}

func (p *InputProcessingPaneBaseData) Identify() ui.PaneID {
	if p.ID == ui.NonePaneID {
		panic("pane has not been assigned an ID")
	}
	return p.ID
}

func (p *InputProcessingPaneBaseData) SetParent(parent ui.FocusQueriable) { p.Parent = parent }

func (p *InputProcessingPaneBaseData) IsVisible() bool { return p.Visible == nil || p.Visible() }
