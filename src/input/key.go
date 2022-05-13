package input

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

type Key struct {
	Mod tcell.ModMask
	Key tcell.Key
	Ch  rune
}

func (k *Key) ToDebugString() string {
	return fmt.Sprintf(
		"(%s (%d),'%s'(%d))",
		tcell.KeyNames[k.Key],
		int(k.Key),
		string(k.Ch),
		int(k.Ch),
	)
}
