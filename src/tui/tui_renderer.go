package tui

import (
	"log"

	"github.com/gdamore/tcell/v2"
)

type TUIRenderer struct {
	screen    tcell.Screen
	needsSync bool
}

func NewTUIRenderer() *TUIRenderer {
	r := &TUIRenderer{}
	r.init()

	return r
}

// Initialize the screen checking errors and return it, so long as no critical
// error occurred.
func (r *TUIRenderer) init() {
	var err error
	r.screen, err = tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	err = r.screen.Init()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	r.screen.SetStyle(defStyle)
	r.screen.EnableMouse()
	r.screen.EnablePaste()
	r.screen.Clear()
}

func (r *TUIRenderer) GetEventPollable() *tcell.Screen {
	return &r.screen
}

func (r *TUIRenderer) Fini() {
	r.screen.Fini()
}

func (r *TUIRenderer) NeedsSync() {
	r.needsSync = true
}

func (r *TUIRenderer) GetScreenDimensions() (int, int) {
	r.screen.SetStyle(tcell.StyleDefault)
	return r.screen.Size()
}

func (r *TUIRenderer) ShowCursor(x, y int) {
	r.screen.ShowCursor(x, y)
}

func (r *TUIRenderer) HideCursor() {
	r.screen.HideCursor()
}

func (r *TUIRenderer) Clear() {
	r.screen.Clear()
}

func (r *TUIRenderer) Show() {
	if r.needsSync {
		r.needsSync = false
		r.screen.Sync()
	} else {
		r.screen.Show()
	}
}

func (t *TUIRenderer) DrawText(x, y, w, h int, style tcell.Style, text string) {
	row := y
	col := x
	for _, r := range text {
		t.screen.SetContent(col, row, r, nil, style)
		col++
		if col >= x+w {
			row++
			col = x
		}
		if row > y+h {
			break
		}
	}
}

func (t *TUIRenderer) DrawBox(style tcell.Style, x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			t.screen.SetContent(col, row, ' ', nil, style)
		}
	}
}
