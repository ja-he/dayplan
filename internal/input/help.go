package input

// Help describes an input-description to action-description mapping.
type Help = map[string]string

// GetHelp returns help for this Tree.
func (t *Tree) GetHelp() Help {
	return t.Root.GetHelp()
}

// GetHelp returns help for this Node.
func (n *Node) GetHelp() Help {
	result := Help{}

	if n.Action != nil {
		result[""] = n.Action.Explain()
	} else {
		for k, c := range n.Children {
			for partialCombo, action := range c.GetHelp() {
				result[ToConfigIdentifierString(k)+partialCombo] = action
			}
		}
	}

	return result
}
