package tui

import (
	"log"

	"github.com/gdamore/tcell/v2"
)

type TUIScreenHandler struct {
	screen    tcell.Screen
	needsSync bool
}

type ConstrainedRenderer interface {
	DrawBox(x, y, w, h int, style tcell.Style)
	DrawText(x, y, w, h int, style tcell.Style, text string)
}

func (r *TUIConstrainedRenderer) Constrain(rawX, rawY, rawW, rawH int) (constrainedX, constrainedY, constrainedW, constrainedH int) {
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

func (r *TUIConstrainedRenderer) DrawText(x, y, w, h int, style tcell.Style, text string) {
	cx, cy, cw, ch := r.Constrain(x, y, w, h)

	r.screenHandler.DrawText(cx, cy, cw, ch, style, text)
}

func (r *TUIConstrainedRenderer) DrawBox(x, y, w, h int, style tcell.Style) {
	cx, cy, cw, ch := r.Constrain(x, y, w, h)
	r.screenHandler.DrawBox(style, cx, cy, cw, ch)
}

type TUIConstrainedRenderer struct {
	screenHandler *TUIScreenHandler

	constraint func() (x, y, w, h int)
}

func NewTUIRenderer() *TUIScreenHandler {
	r := &TUIScreenHandler{}
	r.init()

	return r
}

// Initialize the screen checking errors and return it, so long as no critical
// error occurred.
func (r *TUIScreenHandler) init() {
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

func (r *TUIScreenHandler) GetEventPollable() *tcell.Screen {
	return &r.screen
}

func (r *TUIScreenHandler) Fini() {
	r.screen.Fini()
}

func (r *TUIScreenHandler) NeedsSync() {
	r.needsSync = true
}

// TODO: remove again, probably, antipattern with ui dims being queriable per pane i think
func (r *TUIScreenHandler) GetScreenDimensions() (int, int) {
	r.screen.SetStyle(tcell.StyleDefault)
	return r.screen.Size()
}

func (r *TUIScreenHandler) ShowCursor(x, y int) {
	r.screen.ShowCursor(x, y)
}

func (r *TUIScreenHandler) HideCursor() {
	r.screen.HideCursor()
}

func (r *TUIScreenHandler) Clear() {
	r.screen.Clear()
}

func (r *TUIScreenHandler) Show() {
	if r.needsSync {
		r.needsSync = false
		r.screen.Sync()
	} else {
		r.screen.Show()
	}
}

func (t *TUIScreenHandler) DrawText(x, y, w, h int, style tcell.Style, text string) {
	if w <= 0 || h <= 0 {
		return
	}

	col := x
	row := y
	for _, r := range text {
		t.screen.SetContent(col, row, r, nil, style)
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

func (t *TUIScreenHandler) DrawBox(style tcell.Style, x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			t.screen.SetContent(col, row, ' ', nil, style)
		}
	}
}
