package input

type Action = func()

type Tree struct {
	Root    *Node
	Current *Node
}

func (t *Tree) Process(k Key) (applied bool) {
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

func (t *Tree) Active() bool {
	return t.Current != t.Root
}
