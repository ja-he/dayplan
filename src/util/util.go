package util

import "fmt"

type Rect struct {
	X, Y, W, H int
}

func (r Rect) Contains(x, y int) bool {
	return (x >= r.X) && (x < r.X+r.W) &&
		(y >= r.Y) && (y < r.Y+r.H)
}

// TODO: test and use in appropriate places
func TruncateAt(s string, length int) string {
	r := []rune(s)
	if len(r) <= length {
		return s
	} else {
		return string(append(r[:length-3], []rune("...")...))
	}
}

// Returns a given duration in minutes formatted as a more human-readable
// string of hours and minutes.
func DurationToString(minutes int) string {
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%dh %dmin", hours, mins)
}
