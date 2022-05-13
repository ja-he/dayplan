package input

type Help = map[string]string

func (t *Tree) GetHelp() Help {
	return t.Root.GetHelp()
}

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
