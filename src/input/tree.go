package input

import (
	"fmt"

	"github.com/ja-he/dayplan/src/control/action"
)

type Tree struct {
	Root    *Node
	Current *Node
}

func (t *Tree) ProcessInput(k Key) (applied bool) {
	next := t.Current.Child(k)
	switch {
	case next == nil:
		t.Current = t.Root
		return false
	case next.Action != nil:
		next.Action()
		t.Current = t.Root
		return true
	default:
		t.Current = next
		return true
	}
}

func (t *Tree) CapturesInput() bool {
	return t.Current != t.Root
}

func ConstructInputTree(
	spec map[string]action.Action,
) *Tree {
	root := NewNode()

	for mapping, action := range spec {
		sequence, err := ConfigKeyspecToKeys(mapping)
		if err != nil {
			panic(fmt.Sprintf("error converting config keyspec: '%s'", err.Error()))
		}

		sequenceCurrent := root
		for i, key := range sequence {
			sequenceNext, ok := sequenceCurrent.Children[key]
			if !ok {
				if i == len(sequence)-1 {
					sequenceNext = NewLeaf(action)
				} else {
					sequenceNext = NewNode()
				}
				sequenceCurrent.Children[key] = sequenceNext
			}
			sequenceCurrent = sequenceNext
		}
	}

	return &Tree{
		Root:    root,
		Current: root,
	}
}

func EmptyTree() *Tree {
	root := NewNode()
	return &Tree{
		Root:    root,
		Current: root,
	}
}
