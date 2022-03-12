package tui

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/styling"
)

// ScreenHandler allows rendering to a terminal (via tcell.Screen).
// It also handles synchronization (e.g. on resize) when prompted accordingly.
type ScreenHandler struct {
	screen    tcell.Screen
	needsSync bool
}

// NewTUIScreenHandler initializes and returns a TUIScreenHandler.
func NewTUIScreenHandler() *ScreenHandler {
	r := &ScreenHandler{}
	r.init()

	return r
}

// Initialize the screen checking errors and return it, so long as no critical
// error occurred.
func (s *ScreenHandler) init() {
	var err error
	s.screen, err = tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	err = s.screen.Init()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	s.screen.SetStyle(defStyle)
	s.screen.EnableMouse()
	s.screen.EnablePaste()
	s.screen.Clear()
}

// GetEventPollable returns the underlying screen as an EventPollable.
func (s *ScreenHandler) GetEventPollable() EventPollable {
	return s.screen
}

// Fini finalizes the screen, e.g., for clean program shutdown.
func (s *ScreenHandler) Fini() {
	s.screen.Fini()
}

// NeedsSync registers that a synchronization of the underlying screen is
// necessary.
// This is necessary on resize events.
func (s *ScreenHandler) NeedsSync() {
	s.needsSync = true
}

// GetScreenDimensions returns the current dimensions of the underlying screen.
func (s *ScreenHandler) GetScreenDimensions() (int, int) {
	s.screen.SetStyle(tcell.StyleDefault)
	return s.screen.Size()
}

// ShowCursor sets the position of the text cursor.
func (s *ScreenHandler) ShowCursor(x, y int) {
	s.screen.ShowCursor(x, y)
}

// HideCursor hides the text cursor.
func (s *ScreenHandler) HideCursor() {
	s.screen.HideCursor()
}

// Clear clears the underlying screen.
// If this is not done before drawing new things, old contents that are not
// overwritten will remain visible on the next Show.
func (s *ScreenHandler) Clear() {
	s.screen.Clear()
}

// Show shows the drawn contents, taking the necessity for synchronization into
// account.
func (s *ScreenHandler) Show() {
	if s.needsSync {
		s.needsSync = false
		s.screen.Sync()
	} else {
		s.screen.Show()
	}
}

// DrawText draws given text, within given dimensions in the given style.
func (s *ScreenHandler) DrawText(x, y, w, h int, style tcell.Style, text string) {
	if w <= 0 || h <= 0 {
		return
	}

	col := x
	row := y
	for _, r := range text {
		s.screen.SetContent(col, row, r, nil, style)
		col++
		if col >= x+w {
			row++
			col = x
		}
		if row >= y+h {
			return
		}
	}
}

// DrawBox draws a box of the given dimensions in the given style's background
// color. Note that this overwrites contents within the dimensions.
func (s *ScreenHandler) DrawBox(style tcell.Style, x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			s.screen.SetContent(col, row, ' ', nil, style)
		}
	}
}

// ConstrainedRenderer is a constrained renderer for a TUI.
// It only allows rendering using the underlying screen handler within the set
// dimension constraint.
//
// Non-conforming rendering requests are corrected to be within the bounds.
type ConstrainedRenderer struct {
	screenHandler *ScreenHandler

	constraint func() (x, y, w, h int)
}

// DrawText draws the given text, within the given dimensions, constrained by
// the set constraint, in the given style.
// TODO: should probably change the drawn text manually.
func (r *ConstrainedRenderer) DrawText(x, y, w, h int, styling styling.DrawStyling, text string) {
	cx, cy, cw, ch := r.constrain(x, y, w, h)

	r.screenHandler.DrawText(cx, cy, cw, ch, styling.AsTcell(), text)
}

// DrawBox draws a box of the given dimensions, constrained by the set
// constraint, in the given style.
func (r *ConstrainedRenderer) DrawBox(x, y, w, h int, styling styling.DrawStyling) {
	cx, cy, cw, ch := r.constrain(x, y, w, h)
	r.screenHandler.DrawBox(styling.AsTcell(), cx, cy, cw, ch)
}

func (r *ConstrainedRenderer) constrain(rawX, rawY, rawW, rawH int) (constrainedX, constrainedY, constrainedW, constrainedH int) {
	xConstraint, yConstraint, wConstraint, hConstraint := r.constraint()

	// ensure x, y in bounds, shorten width,height if x,y needed to be moved
	if rawX < xConstraint {
		constrainedX = xConstraint
		rawW -= xConstraint - rawX
	} else {
		constrainedX = rawX
	}
	if rawY < yConstraint {
		constrainedY = yConstraint
		rawH -= yConstraint - rawY
	} else {
		constrainedY = rawY
	}

	xRelativeOffset := constrainedX - xConstraint
	maxAllowableW := wConstraint - xRelativeOffset
	yRelativeOffset := constrainedY - yConstraint
	maxAllowableH := hConstraint - yRelativeOffset

	if rawW > maxAllowableW {
		constrainedW = maxAllowableW
	} else {
		constrainedW = rawW
	}
	if rawH > maxAllowableH {
		constrainedH = maxAllowableH
	} else {
		constrainedH = rawH
	}

	return constrainedX, constrainedY, constrainedW, constrainedH
}

// EventPollable only allows access to PollEvent of a tcell.Screen.
type EventPollable interface {
	PollEvent() tcell.Event
}

// InitializedScreen allows access only to the finalizing functionality of an
// initialized screen.
type InitializedScreen interface {
	Fini()
}

// ScreenSynchronizer allows access only to a screen handler's synchronization
// notification functionality.
type ScreenSynchronizer interface {
	NeedsSync()
}
