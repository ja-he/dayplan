package tui

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/internal/styling"
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

// Dimensions returns the current dimensions of the underlying screen.
func (s *ScreenHandler) Dimensions() (x, y, w, h int) {
	w, h = s.screen.Size()
	return 0, 0, w, h
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
func (s *ScreenHandler) DrawText(x, y, w, h int, style styling.DrawStyling, text string) {
	if w <= 0 || h <= 0 {
		return
	}

	tcellStyle := style.AsTcell()

	col := x
	row := y
	for _, r := range text {
		s.screen.SetContent(col, row, r, nil, tcellStyle)
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
func (s *ScreenHandler) DrawBox(x, y, w, h int, style styling.DrawStyling) {
	tcellStyle := style.AsTcell()
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			s.screen.SetContent(col, row, ' ', nil, tcellStyle)
		}
	}
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
