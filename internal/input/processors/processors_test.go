package processors_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/internal/control/action"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/input/processors"
)

func TestModalInputProcessor(t *testing.T) {

	t.Run("CapturesInput", func(t *testing.T) {
		dummy := dummySIP{captures: false}
		m := processors.NewModalInputProcessor(&dummy)
		if m.CapturesInput() {
			t.Error("claims to capture input, initially")
		}
		dummy.captures = true
		if !m.CapturesInput() {
			t.Error("fails to (claim to) capture input, despite its simple processor doing so")
		}
		m.ApplyModalOverlay(&dummySIP{captures: false})
		if m.CapturesInput() {
			t.Error("claims to capture input, despite its overlay not capturing")
		}
		dummy.captures = false
		m.ApplyModalOverlay(&dummySIP{captures: true})
		m.ApplyModalOverlay(&dummySIP{captures: false})
		if m.CapturesInput() {
			t.Error("claims to capture input, despite its topmost overlay not capturing")
		}
	})

	t.Run("ProcessInput", func(t *testing.T) {
		x := input.Key{Key: tcell.KeyRune, Ch: 'x'}
		y := input.Key{Key: tcell.KeyRune, Ch: 'y'}
		dummy := dummySIP{inputs: map[input.Key]bool{
			x: true,
			y: false,
		}}
		m := processors.NewModalInputProcessor(&dummy)
		if !m.ProcessInput(x) {
			t.Error("does not process x")
		}
		if m.ProcessInput(y) {
			t.Error("processes y")
		}
	})

	t.Run("{Apply,Pop}ModalOverlay(s)", func(t *testing.T) {
		x := input.Key{Key: tcell.KeyRune, Ch: 'x'}
		y := input.Key{Key: tcell.KeyRune, Ch: 'y'}
		z := input.Key{Key: tcell.KeyRune, Ch: 'z'}
		a := dummySIP{inputs: map[input.Key]bool{
			x: true,
			y: false,
			z: false,
		}}
		b := dummySIP{inputs: map[input.Key]bool{
			x: false,
			y: true,
			z: false,
		}}
		c := dummySIP{inputs: map[input.Key]bool{
			x: false,
			y: false,
			z: true,
		}}
		m := processors.NewModalInputProcessor(&a)

		checkProcessesTuple := func(msg string, expected [3]bool) {
			actual := [3]bool{
				m.ProcessInput(x),
				m.ProcessInput(y),
				m.ProcessInput(z),
			}
			if actual != expected {
				t.Error(msg, "check failed:", expected, "!=", actual)
			}
		}
		assumeIndex := func(actual uint, expected uint) {
			if actual != expected {
				t.Errorf("got overlay index %d instead of %d", actual, expected)
			}
		}
		assumeNil := func(e error) {
			if e != nil {
				t.Error("unexpected non-nil on assumed-valid overlay pop:", e.Error())
			}
		}
		assumeErr := func(e error) {
			if e == nil {
				t.Error("unexpected nil error on assumed-invalid overlay pop")
			}
		}

		checkProcessesTuple("base", [3]bool{true, false, false})

		assumeIndex(m.ApplyModalOverlay(&b), 0)
		checkProcessesTuple("overlay1", [3]bool{false, true, false})

		assumeIndex(m.ApplyModalOverlay(&c), 1)
		checkProcessesTuple("overlay2", [3]bool{false, false, true})

		assumeNil(m.PopModalOverlay())
		checkProcessesTuple("overlay1", [3]bool{false, true, false})

		assumeNil(m.PopModalOverlay())
		checkProcessesTuple("base", [3]bool{true, false, false})

		assumeErr(m.PopModalOverlay())

		assumeIndex(m.ApplyModalOverlay(&b), 0)
		assumeIndex(m.ApplyModalOverlay(&c), 1)
		assumeIndex(m.ApplyModalOverlay(&b), 2)
		assumeIndex(m.ApplyModalOverlay(&b), 3)
		assumeIndex(m.ApplyModalOverlay(&b), 4)
		assumeIndex(m.ApplyModalOverlay(&a), 5)
		checkProcessesTuple("a topmost", [3]bool{true, false, false})

		// this should be a no-op
		m.PopModalOverlays(1000)
		checkProcessesTuple("a topmost still", [3]bool{true, false, false})

		// pop down so that c at 1 is on top
		m.PopModalOverlays(2)
		checkProcessesTuple("c at 1", [3]bool{false, false, true})

		// pop down so that b at 0 is on top
		m.PopModalOverlays(1)
		checkProcessesTuple("b at 0", [3]bool{false, true, false})

		// pop down back to base
		m.PopModalOverlays(0)
		checkProcessesTuple("a at base", [3]bool{true, false, false})

		// this should be a no-op
		m.PopModalOverlays(1000)
		checkProcessesTuple("a at base, still", [3]bool{true, false, false})
	})

	t.Run("GetHelp", func(t *testing.T) {
		a := dummySIP{help: map[string]string{}}
		m := processors.NewModalInputProcessor(&a)
		if len(m.GetHelp()) != 0 {
			t.Error("help on empty not empty")
		}

		a.help["asdf"] = "qwer"
		help := m.GetHelp()
		if len(help) != 1 || help["asdf"] != "qwer" {
			t.Error("help looks unexpected:", help)
		}

		b := dummySIP{help: map[string]string{}}
		m.ApplyModalOverlay(&b)
		help = m.GetHelp()
		if len(help) != 0 {
			t.Error("overlay help looks unexpected:", help)
		}

		b.help["foo"] = "bar"
		b.help["nucleus"] = "alleycat"
		help = m.GetHelp()
		if len(help) != 2 || help["foo"] != "bar" || help["nucleus"] != "alleycat" {
			t.Error("overlay help looks unexpected:", help)
		}
	})

}

func TestTextInputProcessor(t *testing.T) {
	cA := input.Key{Key: tcell.KeyCtrlA}
	cB := input.Key{Key: tcell.KeyCtrlB}
	cY := input.Key{Key: tcell.KeyCtrlY}
	x := input.Key{Key: tcell.KeyRune, Ch: 'x'}
	y := input.Key{Key: tcell.KeyRune, Ch: 'y'}
	z := input.Key{Key: tcell.KeyRune, Ch: 'z'}

	t.Run("ProcessInput", func(t *testing.T) {

		t.Run("runes", func(t *testing.T) {
			r := rune(0)
			callback := func(newRune rune) { r = newRune }
			p := processors.NewTextInputProcessor(
				map[input.Key]action.Action{
					cY: &dummyAction{action: func() { t.Error("the cY callback was called, which it should not have been") }},
				},
				callback,
			)

			p.ProcessInput(x)
			if r != 'x' {
				t.Error("rune was not set to x but is", r)
			}
			p.ProcessInput(y)
			if r != 'y' {
				t.Error("rune was not set to y but is", r)
			}
			p.ProcessInput(z)
			if r != 'z' {
				t.Error("rune was not set to z but is", r)
			}
		})

		t.Run("specials", func(t *testing.T) {
			cACalled := false
			cBCalled := false
			p := processors.NewTextInputProcessor(
				map[input.Key]action.Action{
					cA: &dummyAction{action: func() { cACalled = true }},
					cB: &dummyAction{action: func() { cBCalled = true }},
				},
				func(rune) {},
			)

			if !p.ProcessInput(cA) || !cACalled {
				t.Error("action for <c-a> not done")
			}
			if !p.ProcessInput(cB) || !cBCalled {
				t.Error("action for <c-a> not done")
			}
			if p.ProcessInput(cY) {
				t.Error("claims to apply <c-y> with no such mapping")
			}
		})

	})

	t.Run("CapturesInput", func(t *testing.T) {
		p := processors.NewTextInputProcessor(map[input.Key]action.Action{}, func(rune) {})
		if !p.CapturesInput() {
			t.Error("text input processor does not unconditionally capture input")
		}
	})

	t.Run("GetHelp", func(t *testing.T) {
		p := processors.NewTextInputProcessor(
			map[input.Key]action.Action{
				cA: &dummyAction{explanation: "Aaa"},
				cB: &dummyAction{explanation: "Bbb"},
			},
			func(rune) {},
		)
		help := p.GetHelp()
		if !(len(help) == 2 && help[input.ToConfigIdentifierString(cA)] == "Aaa" && help[input.ToConfigIdentifierString(cB)] == "Bbb") {
			t.Error("help looks unexpected:", help)
		}

		p = processors.NewTextInputProcessor(
			map[input.Key]action.Action{},
			func(rune) {},
		)
		if len(p.GetHelp()) != 0 {
			t.Error("help on empty not empty")
		}
	})

}

// dummy simple input processor for testing
type dummySIP struct {
	captures bool
	inputs   map[input.Key]bool
	help     map[string]string
}

func (d *dummySIP) CapturesInput() bool           { return d.captures }
func (d *dummySIP) ProcessInput(k input.Key) bool { return d.inputs[k] }
func (d *dummySIP) GetHelp() map[string]string    { return d.help }

// dummy action for testing
type dummyAction struct {
	action      func()
	explanation string
}

func (d *dummyAction) Do()             { d.action() }
func (d *dummyAction) Undo()           {}
func (d *dummyAction) Undoable() bool  { return false }
func (d *dummyAction) Explain() string { return d.explanation }
