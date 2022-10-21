package action

// Action models a "doable" and potentially "undoable" action.
//
// If an action can be undone, its Undoable member is to return true, otherwise
// false. Iff Undoable returns false, Undo should be a no-op.
//
// Actions can also -- e.g. for the purposes of help generation -- be described
// by Explain. Explain is to return a string describing the action Do would perform.
type Action interface {
	Do()

	Undo()
	Undoable() bool

	Explain() string
}
