package util

type Rect struct {
	X, Y, W, H int
}

func (r Rect) Contains(x, y int) bool {
	return (x >= r.X) && (x < r.X+r.W) &&
		(y >= r.Y) && (y < r.Y+r.H)
}
