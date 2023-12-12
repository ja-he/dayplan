package ui

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// CursorLocationRequestHandler is an interface for a type that can handle
// requests to place a (text/terminal) cursor on the screen.
type CursorLocationRequestHandler interface {
	Put(l CursorLocation, requesterID string)
	Delete(requesterID string)
}

// CursorWrangler handles requests to place a (text/terminal) cursor on the
// screen.
type CursorWrangler struct {
	mtx sync.RWMutex

	cc TextCursorController

	desiredLocation     *CursorLocation
	mostRecentRequester *string
}

// NewCursorWrangler creates a new CursorWrangler.
func NewCursorWrangler(controller TextCursorController) *CursorWrangler {
	return &CursorWrangler{
		cc:                  controller,
		desiredLocation:     nil,
		mostRecentRequester: nil,
	}
}

// Put places the cursor at the given location.
func (w *CursorWrangler) Put(l CursorLocation, requesterID string) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	if w.desiredLocation != nil && *w.mostRecentRequester != requesterID {
		log.Warn().Msgf("being asked to put cursor (at %s) while it is already placed by '%s' (at %s); will be overwritten", l.String(), *w.mostRecentRequester, w.desiredLocation.String())
	}

	w.desiredLocation = &l
	w.mostRecentRequester = &requesterID
}

// Delete removes the cursor.
func (w *CursorWrangler) Delete(requesterID string) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	if w.mostRecentRequester == nil {
		log.Debug().Msgf("ignoring '%s's request to delete cursor, as there is no active request for a cursor (i.e., the requester's intention is already met, another requester had already superseded their request)", requesterID)
		return
	}

	if *w.mostRecentRequester != requesterID {
		log.Debug().Msgf("ignoring '%s's request to delete cursor, as current requestor is %s", requesterID, *w.mostRecentRequester)
		return
	}

	w.desiredLocation = nil
	w.mostRecentRequester = nil
}

// Enact enacts the current cursor location request via the underlying
// cursor controller.
func (w *CursorWrangler) Enact() {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	if w.desiredLocation != nil {
		w.cc.ShowCursor(*w.desiredLocation)
	} else {
		w.cc.HideCursor()
	}
}
