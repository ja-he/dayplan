package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/ui"
)

type InputProcessingPaneBaseData struct {
	ID             ui.PaneID
	Parent         ui.FocusQueriable
	InputProcessor input.ModalInputProcessor
}

func (p *InputProcessingPaneBaseData) Identify() ui.PaneID { return p.ID }

func (p *InputProcessingPaneBaseData) SetParent(parent ui.FocusQueriable) { p.Parent = parent }
