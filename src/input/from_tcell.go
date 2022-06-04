package input

import "github.com/gdamore/tcell/v2"

// KeyFromTcellEvent formats a tcell.EventKey to a Key as this package expects
// it. Any Key for a tcell.EventKey should be converted by this function.
func KeyFromTcellEvent(e *tcell.EventKey) Key {
	if e.Key() == tcell.KeyRune {
		return Key{Key: tcell.KeyRune, Ch: e.Rune()}
	} else {
		return Key{Key: e.Key()}
	}
}
