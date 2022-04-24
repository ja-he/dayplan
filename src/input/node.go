package input

import (
	"github.com/ja-he/dayplan/src/control/action"
)

type Node struct {
	Children map[Key]*Node
	Action   action.Action
}

// Child returns the child node for the given Key.
// Returns nil, if there is no child node for the key.
func (n *Node) Child(k Key) (child *Node) {
	child, ok := n.Children[k]
	if ok {
		return child
	} else {
		return nil
	}
}

func NewNode() *Node {
	return &Node{
		Children: make(map[Key]*Node),
	}
}

func NewLeaf(action action.Action) *Node {
	return &Node{
		Action: action,
	}
}
