package action

// Simple implements the Action interface.
// It models a simple, non-undoable action as a func() which is called on Do.
type Simple struct {
	action  func()
	explain func() string
}

// Do performs this simple action.
// A simple action is not undoable.
func (a *Simple) Do() {
	a.action()
}

// Undoable always returns false.
// A simple action is not undoable.
func (a *Simple) Undoable() bool { return false }

// Undo does nothing.
// A simple action is not undoable.
func (a *Simple) Undo() {}

// Explain returns the explanation for this simple action's Do member.
func (a *Simple) Explain() string {
	return a.explain()
}

// NewSimple returns a pointer to a new simple action, which stores the given
// action function and the given explainer to use when prompted with Do or
// Explain respectively.
func NewSimple(explainer func() string, action func()) *Simple {
	return &Simple{
		action:  action,
		explain: explainer,
	}
}
