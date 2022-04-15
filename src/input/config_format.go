package input

import (
	"fmt"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

func ConfigKeyspecToKeys(spec string) ([]Key, error) {
	specR := []rune(spec)
	keys := make([][]rune, 0)
	specialContext := false

	for pos := range spec {
		switch spec[pos] {

		case '<':
			if specialContext {
				return []Key{}, fmt.Errorf("illegal second opening special context ('<') before previous is closed (pos %d)", pos)
			}
			specialContext = true
			keys = append(keys, []rune{specR[pos]})

		case '>':
			if !specialContext {
				return []Key{}, fmt.Errorf("illegal closing of special context ('>') while none open (pos %d)", pos)
			}
			specialContext = false
			keys[len(keys)-1] = append(keys[len(keys)-1], specR[pos])

		default:
			if specialContext {
				if !unicode.IsLetter(specR[pos]) && spec[pos] != '-' {
					return []Key{},
						fmt.Errorf("illegal character '%c' in special context (pos %d)", spec[pos], pos)
				}
				keys[len(keys)-1] = append(keys[len(keys)-1], specR[pos])
			} else {
				keys = append(keys, []rune{specR[pos]})
			}

		}
	}

	result := make([]Key, 0)
	for _, keyIdentifier := range keys {
		if keyIdentifier[0] == '<' {
			key, err := KeyIdentifierToKey(string(keyIdentifier[1 : len(keyIdentifier)-1]))
			if err != nil {
				return []Key{}, fmt.Errorf("error mapping identifier '%s' to key: %s", string(keyIdentifier), err.Error())
			}
			result = append(result, key)
		} else {
			result = append(result, Key{Key: tcell.KeyRune, Ch: keyIdentifier[0]})
		}
	}

	return result, nil
}

func KeyIdentifierToKey(identifier string) (Key, error) {
	mapping := map[string]Key{
		"space": {Key: tcell.KeyRune, Ch: ' '},
		"cr":    {Key: tcell.KeyEnter},
		"esc":   {Key: tcell.KeyESC},
		"del":   {Key: tcell.KeyDelete},
		"bs":    {Key: tcell.KeyBackspace2},
		"left":  {Key: tcell.KeyLeft},
		"right": {Key: tcell.KeyRight},

		"c-space": {Key: tcell.KeyCtrlSpace},
		"c-bs":    {Key: tcell.KeyBackspace},

		"c-a": {Key: tcell.KeyCtrlA},
		"c-b": {Key: tcell.KeyCtrlB},
		"c-c": {Key: tcell.KeyCtrlC},
		"c-d": {Key: tcell.KeyCtrlD},
		"c-e": {Key: tcell.KeyCtrlE},
		"c-f": {Key: tcell.KeyCtrlF},
		"c-g": {Key: tcell.KeyCtrlG},
		"c-h": {Key: tcell.KeyCtrlH},
		"c-i": {Key: tcell.KeyCtrlI},
		"c-j": {Key: tcell.KeyCtrlJ},
		"c-k": {Key: tcell.KeyCtrlK},
		"c-l": {Key: tcell.KeyCtrlL},
		"c-m": {Key: tcell.KeyCtrlM},
		"c-n": {Key: tcell.KeyCtrlN},
		"c-o": {Key: tcell.KeyCtrlO},
		"c-p": {Key: tcell.KeyCtrlP},
		"c-q": {Key: tcell.KeyCtrlQ},
		"c-r": {Key: tcell.KeyCtrlR},
		"c-s": {Key: tcell.KeyCtrlS},
		"c-t": {Key: tcell.KeyCtrlT},
		"c-u": {Key: tcell.KeyCtrlU},
		"c-v": {Key: tcell.KeyCtrlV},
		"c-w": {Key: tcell.KeyCtrlW},
		"c-x": {Key: tcell.KeyCtrlX},
		"c-y": {Key: tcell.KeyCtrlY},
		"c-z": {Key: tcell.KeyCtrlZ},
	}

	key, ok := mapping[identifier]
	if !ok {
		return Key{}, fmt.Errorf("no mapping present for identifier '%s'", identifier)
	}

	return key, nil
}
