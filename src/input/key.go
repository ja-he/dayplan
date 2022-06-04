package input

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// Key represents a key input.
//
// NOTE:
//   currently barely generifying tcell's input type; could eventually be
//   properly generified for other input sources.
type Key struct {
	Mod tcell.ModMask
	Key tcell.Key
	Ch  rune
}

// ToDebugString returns a debug string for this key.
func (k *Key) ToDebugString() string {
	return fmt.Sprintf(
		"(%s (%d),'%s'(%d))",
		tcell.KeyNames[k.Key],
		int(k.Key),
		string(k.Ch),
		int(k.Ch),
	)
}
