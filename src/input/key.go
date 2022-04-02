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

func (k *Key) ToString() string {
	return fmt.Sprintf(
		"(%s,'%s'(%d))",
		tcell.KeyNames[k.Key],
		string(k.Ch),
		int(k.Ch),
	)
}
