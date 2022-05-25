package input_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"

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

// to avoid depending on 'action' functions
type DummyAction struct{}

func (d *DummyAction) Do()             {}
func (d *DummyAction) Undo()           {}
func (d *DummyAction) Undoable() bool  { return false }
func (d *DummyAction) Explain() string { return "dummy" }
