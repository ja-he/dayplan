package action

type Simple struct {
	action func()

	explain func() string
}

func (a *Simple) Do() {
	a.action()
}

func (a *Simple) Undoable() bool { return false }

func (a *Simple) Undo() {}

func (a *Simple) Explain() string {
	return a.explain()
}

func NewSimple(explainer func() string, action func()) *Simple {
	return &Simple{
		action:  action,
		explain: explainer,
	}
}
