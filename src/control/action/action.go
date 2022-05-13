package action

type Action interface {
	Do()

	Undo()
	Undoable() bool

	Explain() string
}
