package input_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/ja-he/dayplan/src/control/action"
	"github.com/ja-he/dayplan/src/input"
)

func TestNewNode(t *testing.T) {
	n := input.NewNode()
	if n.Children == nil {
		t.Error("node.Children not initialized")
	}
	if len(n.Children) != 0 {
		t.Error("node.Children not empty")
	}
}

func TestNewLeaf(t *testing.T) {
	a := DummyAction{}
	l := input.NewLeaf(&a)
	if l.Action != &a {
		t.Error("action not assigned properly to leaf")
	}
	if !(l.Children == nil || len(l.Children) == 0) {
		t.Error("expected leaf to have nil children or to be empty")
	}
}

func TestChild(t *testing.T) {
	t.Run("node with no child gives no child", func(t *testing.T) {
		n := input.NewNode()
		child := n.Child(input.Key{Key: tcell.KeyRune, Ch: 'x'})
		if child != nil {
			t.Errorf("given non-nil child %#v for non-entered input", child)
		}
		if n.Action != nil {
			t.Error("expected new node to have nil action")
		}
	})
	t.Run("node with leaf child on x gives leaf for Child(x)", func(t *testing.T) {
		key := input.Key{Key: tcell.KeyRune, Ch: 'x'}
		action := DummyAction{}
		leaf := input.NewLeaf(&action)

		n := input.NewNode()
		n.Children[key] = leaf
		child := n.Child(key)
		if child == nil {
			t.Error("given nil child for mapped input")
		}
		if child != leaf {
			t.Errorf("not given expected leaf, but %#v", child)
		}
	})
}

func TestConstructInputTree(t *testing.T) {

	t.Run("empty map produces single-node tree", func(t *testing.T) {
		emptyTree, err := input.ConstructInputTree(make(map[string]action.Action))
		if err != nil {
			t.Error(err.Error())
		}
		validateNewlyCreatedTree(t, emptyTree)
		if !(emptyTree.Root.Children != nil && len(emptyTree.Root.Children) == 0) {
			t.Error("empty tree's root node should be the only one, but has children:", emptyTree.Root.Children)
		}
		if emptyTree.ProcessInput(input.Key{Key: tcell.KeyRune, Ch: 'x'}) {
			t.Error("empty tree claims to apply (non-added) input")
		}
	})

	t.Run("single input sequence", func(t *testing.T) {
		shouldGetSetToTrue := false
		tree, err := input.ConstructInputTree(map[string]action.Action{"xyz": &DummyAction{F: func() { shouldGetSetToTrue = true }}})
		if err != nil {
			t.Error(err.Error())
		}
		validateNewlyCreatedTree(t, tree)
		if tree.ProcessInput(input.Key{}) {
			t.Error("tree processes non-added input")
		}
		if tree.CapturesInput() {
			t.Error("tree claims to capture input after processing non-added")
		}

		if !tree.ProcessInput(input.Key{Key: tcell.KeyRune, Ch: 'x'}) {
			t.Error("tree fails to process added input")
		}
		if !tree.CapturesInput() {
			t.Error("tree fails to capture input in the middle of a sequence")
		}
		if !tree.ProcessInput(input.Key{Key: tcell.KeyRune, Ch: 'y'}) {
			t.Error("tree fails to process added input")
		}
		if !tree.CapturesInput() {
			t.Error("tree fails to capture input in the middle of a sequence")
		}
		if !tree.ProcessInput(input.Key{Key: tcell.KeyRune, Ch: 'z'}) {
			t.Error("tree fails to process added input")
		}
		if !shouldGetSetToTrue {
			t.Error("action not applied")
		}
		if tree.CapturesInput() {
			t.Error("tree claims to capture input after complete sequence")
		}
	})

	t.Run("complex inputs", func(t *testing.T) {
		xyzTrueable := false
		ctrlaTrueable := false
		tree, err := input.ConstructInputTree(map[string]action.Action{
			"xyz":   &DummyAction{F: func() { xyzTrueable = true }},
			"<c-a>": &DummyAction{F: func() { ctrlaTrueable = true }},
		})
		if err != nil {
			t.Error(err.Error())
		}
		validateNewlyCreatedTree(t, tree)
		if tree.ProcessInput(input.Key{}) {
			t.Error("tree processes non-added input")
		}
		if tree.CapturesInput() {
			t.Error("tree claims to capture input after processing non-added")
		}

		if !tree.ProcessInput(input.Key{Key: tcell.KeyRune, Ch: 'x'}) {
			t.Error("tree fails to process added input")
		}
		if !tree.CapturesInput() {
			t.Error("tree fails to capture input in the middle of a sequence")
		}
		if tree.ProcessInput(input.Key{}) {
			t.Error("tree processes invalid input in middle of sequence")
		}
		if !tree.ProcessInput(input.Key{Key: tcell.KeyCtrlA}) {
			t.Error("tree fails to process input <c-a>")
		}
		if !ctrlaTrueable {
			t.Error("action not applied")
		}
		if tree.CapturesInput() {
			t.Error("tree claims to capture input after sequence")
		}

		if !tree.ProcessInput(input.Key{Key: tcell.KeyRune, Ch: 'x'}) {
			t.Error("tree fails to process added input")
		}
		if !tree.CapturesInput() {
			t.Error("tree fails to capture input in the middle of a sequence")
		}
		if !tree.ProcessInput(input.Key{Key: tcell.KeyRune, Ch: 'y'}) {
			t.Error("tree fails to process added input")
		}
		if !tree.CapturesInput() {
			t.Error("tree fails to capture input in the middle of a sequence")
		}
		if !tree.ProcessInput(input.Key{Key: tcell.KeyRune, Ch: 'z'}) {
			t.Error("tree fails to process added input")
		}
		if !xyzTrueable {
			t.Error("action not applied")
		}
		if tree.CapturesInput() {
			t.Error("tree claims to capture input after complete sequence")
		}
	})

	t.Run("invalid keyspec errors", func(t *testing.T) {
		tree, err := input.ConstructInputTree(map[string]action.Action{"<asdf": &DummyAction{}})
		if err == nil {
			t.Error("nil error despite invalid keyspec")
		}
		if tree != nil {
			t.Error("non-nil tree despite invalid keyspec")
		}
	})

}

func TestEmptyTree(t *testing.T) {
	tree := input.EmptyTree()
	validateNewlyCreatedTree(t, tree)
}

func validateNewlyCreatedTree(t *testing.T, newlyCreated *input.Tree) {
	t.Helper() // NOTE(ja_he): Almost certainly not needed

	if newlyCreated.Root == nil || newlyCreated.Current == nil {
		t.Error("either root or current is nil on newly created tree:", newlyCreated.Root, ",", newlyCreated.Current)
	}
	if newlyCreated.Root != newlyCreated.Current {
		t.Error("root and current differ on newly created tree:", newlyCreated.Root, ",", newlyCreated.Current)
	}
	if newlyCreated.CapturesInput() {
		t.Error("newly created tree claims to capture input")
	}
}

// to avoid depending on 'action' functions
type DummyAction struct {
	F func()
}

func (d *DummyAction) Do()             { d.F() }
func (d *DummyAction) Undo()           {}
func (d *DummyAction) Undoable() bool  { return false }
func (d *DummyAction) Explain() string { return "dummy" }
