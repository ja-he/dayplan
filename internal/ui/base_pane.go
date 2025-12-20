package ui

import (
	"path"
	"strings"

	"github.com/ja-he/dayplan/internal/input"
	"github.com/rs/zerolog/log"
)

// BasePane is the base data necessary for a UI pane and provides a base
// implementation using them.
//
// Note that constructing this value that you need to assign the ID.
type BasePane struct {
	ID string

	Parent   PaneQuerier
	Children []PaneQuerier

	InputProcessor input.ModalInputProcessor

	Visible func() bool
}

// Identify returns the panes ID.
func (p *BasePane) Identify() string { return p.ID }

// SetParent sets the pane's parent.
func (p *BasePane) SetParent(parent PaneQuerier) { p.Parent = parent }

// IsVisible indicates whether the pane is visible.
func (p *BasePane) IsVisible() bool { return p.Visible == nil || p.Visible() }

func (p *BasePane) GetChild(pathToChild string) PaneQuerier {
	log.Debug().Msgf("Asked for child '%s'", pathToChild)

	if path.IsAbs(pathToChild) {
		if p.Parent != nil {
			log.Error().Msgf("Pane '%s' with parent '%s' asked for absolute child path '%s' despite having a parent, thus not being root.", p.ID, p.Parent.Identify(), pathToChild)
			return nil
		}
		// When the path is absolute, as it is here, it starts with '/' therefore
		// we know that SplitN(..., 2) will return 2 elements, the first being the
		// empty string.
		return p.GetChild(strings.SplitN(pathToChild, "/", 2)[1])
	}

	pathSplit := strings.SplitN(pathToChild, "/", 2)
	log.Trace().Msgf("Will check %d children for '%s'", len(p.Children), pathSplit[0])
	for _, c := range p.Children {
		log.Trace().Msgf("Checking child '%s'", c.Identify())
		if c.Identify() == pathSplit[0] {
			if len(pathSplit) > 1 {
				log.Trace().Msgf("Deferring child request to '%s'", c.Identify())
				return c.GetChild(pathSplit[1])
			}
			return c
		}
	}

	return nil
}
