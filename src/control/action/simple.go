package action

type Simple struct {
	action func()
}

func (a *Simple) Do() {
	a.action()
}

func (a *Simple) Undoable() bool { return false }

func (a *Simple) Undo() {}

func NewSimple(action func()) *Simple {
	return &Simple{
		action: action,
	}
}
