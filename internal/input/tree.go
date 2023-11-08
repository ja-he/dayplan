package input

import (
	"fmt"

	"github.com/ja-he/dayplan/internal/control/action"
)

// Tree represents an input tree, which can contain various input sequences
// that terminate in an action.
//
// Example:
//
//	tree:                       mapping:
//
//	x
//	+-y
//	| +-z   -> action1          "xyz" -> action1
//	+-z     -> action2          "xz"  -> action2
//	z       -> action3          "z"   -> action3
type Tree struct {
	Root    *Node
	Current *Node
}

// ProcessInput attempts to process the provided input.
// Returns whether the provided input "applied", i.E. the processor performed
// an action based on the input.
func (t *Tree) ProcessInput(k Key) (applied bool) {
	next := t.Current.Child(k)
	switch {
	case next == nil:
		t.Current = t.Root
		return false
	case next.Action != nil:
		next.Action.Do()
		t.Current = t.Root
		return true
	default:
		t.Current = next
		return true
	}
}

// CapturesInput returns whether this processor "captures" input, i.E. whether
// it ought to take priority in processing over other processors.
// This is useful, e.g., for prioritizing processors whith partial input
// sequences or for such overlays, that are to take complete priority by
// completely gobbling all input.
func (t *Tree) CapturesInput() bool {
	return t.Current != t.Root
}

// ConstructInputTree construct a Tree for the given mappings of input
// sequence strings to actions.
// If the given mapping is invalid, this returns an error.
func ConstructInputTree(
	spec map[Keyspec]action.Action,
) (*Tree, error) {
	root := NewNode()

	for mapping, action := range spec {
		sequence, err := ConfigKeyspecToKeys(mapping)
		if err != nil {
			return nil, fmt.Errorf("error converting config keyspec: '%s'", err.Error())
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
	}, nil
}

// EmptyTree returns a pointer to an empty tree.
func EmptyTree() *Tree {
	root := NewNode()
	return &Tree{
		Root:    root,
		Current: root,
	}
}
