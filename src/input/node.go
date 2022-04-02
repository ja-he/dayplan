package input

type Node struct {
	Children map[Key]*Node
	Action   Action
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
