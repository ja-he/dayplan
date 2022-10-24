package input

import (
	"github.com/ja-he/dayplan/internal/control/action"
)

// Node is a node in a Tree.
// It can have child nodes or an action.
//
// NOTE:
//
//	must not have children if it has an action, and must not have an action if
//	it has children.
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

// NewNode returns a pointer to a new empty node Node with initialized children.
//
// NOTE: to construct a leaf with an action, prefer NewLeaf.
func NewNode() *Node {
	return &Node{
		Children: make(map[Key]*Node),
	}
}

// NewLeaf returns a pointer to a new action leaf Node without children.
//
// NOTE: to construct an intermediate node with no aciton, prefer NewNode.
func NewLeaf(action action.Action) *Node {
	return &Node{
		Action: action,
	}
}
