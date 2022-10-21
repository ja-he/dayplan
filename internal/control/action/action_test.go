package action_test

import (
	"testing"

	"github.com/ja-he/dayplan/internal/control/action"
)

func TestSimpleInterface(t *testing.T) {

	t.Run("Do(/Undo)", func(t *testing.T) {
		becomesTrue := false
		a := func() { becomesTrue = true }

		s := action.NewSimple(func() string { return "sets flag to true" }, a)
		s.Do()

		if !becomesTrue {
			t.Error("action was not executed properly (flag unchanged)")
		}

		// just for coverage, honestly
		s.Undo()
		if !becomesTrue {
			t.Error("undo did something?!")
		}
	})

	t.Run("Undoable", func(t *testing.T) {
		s := action.NewSimple(func() string { return "does nothing" }, func() {})
		if s.Undoable() {
			t.Error("simple action claims to be undoable")
		}
	})

	t.Run("Explain", func(t *testing.T) {
		e := "does nothing"
		s := action.NewSimple(func() string { return e }, func() {})
		if s.Explain() != "does nothing" {
			t.Error("initial explanation wrong:", s.Explain())
		}
		e = "does nothing, very well"
		if s.Explain() != "does nothing, very well" {
			t.Error("changed explanation wrong:", s.Explain())
		}
	})

}
