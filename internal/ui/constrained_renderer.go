package ui

import "github.com/ja-he/dayplan/internal/styling"

// CR is a constrained renderer for a TUI.
// It only allows rendering using the underlying screen handler within the set
// dimension constraint.
//
// Non-conforming rendering requests are corrected to be within the bounds.
type CR struct {
	renderer Renderer

	constraint func() (x, y, w, h int)
}

func NewConstrainedRenderer(
	renderer ConstrainedRenderer,
	constraint func() (x, y, w, h int),
) *CR {
	return &CR{
		renderer:   renderer,
		constraint: constraint,
	}
}

func (r *CR) Dimensions() (x, y, w, h int) {
	return r.constraint()
}

// DrawText draws the given text, within the given dimensions, constrained by
// the set constraint, in the given style.
// TODO: should probably change the drawn text manually.
func (r *CR) DrawText(x, y, w, h int, styling styling.DrawStyling, text string) {
	cx, cy, cw, ch := r.constrain(x, y, w, h)

	r.renderer.DrawText(cx, cy, cw, ch, styling, text)
}

// DrawBox draws a box of the given dimensions, constrained by the set
// constraint, in the given style.
func (r *CR) DrawBox(x, y, w, h int, sty styling.DrawStyling) {
	cx, cy, cw, ch := r.constrain(x, y, w, h)
	r.renderer.DrawBox(cx, cy, cw, ch, sty)
}

func (r *CR) constrain(rawX, rawY, rawW, rawH int) (constrainedX, constrainedY, constrainedW, constrainedH int) {
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
