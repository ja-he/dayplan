package input

import "github.com/gdamore/tcell/v2"

func KeyFromTcellEvent(e *tcell.EventKey) Key {
	if e.Key() == tcell.KeyRune {
		return Key{Key: tcell.KeyRune, Ch: e.Rune()}
	} else {
		return Key{Key: e.Key()}
	}
}
