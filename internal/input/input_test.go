package input_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/ja-he/dayplan/internal/control/action"
	"github.com/ja-he/dayplan/internal/input"
)

func TestConfigKeyspecToKey(t *testing.T) {

	t.Run("valid", func(t *testing.T) {
		expectValid := func(s input.Keyspec) []input.Key {
			keys, err := input.ConfigKeyspecToKeys(s)
			if err != nil {
				t.Error("unexpected error on valid spec:", err.Error())
			}
			if keys == nil {
				t.Error("unexpected nil keyspec on valid spec")
			}
			return keys
		}

		t.Run("empty", func(t *testing.T) {
			keys := expectValid("")
			if len(keys) != 0 {
				t.Error("expected empty seq of keys")
			}
		})

		t.Run("single", func(t *testing.T) {
			keys := expectValid("x")
			if len(keys) != 1 {
				t.Error("expected single key")
			}
			if (keys[0] != input.Key{Key: tcell.KeyRune, Ch: 'x'}) {
				t.Error("expected single key to be 'x'")
			}
		})

		t.Run("special", func(t *testing.T) {
			t.Run("<c-a>", func(t *testing.T) {
				keys := expectValid("<c-a>")
				if len(keys) != 1 {
					t.Error("expected single key")
				}
				if (keys[0] != input.Key{Key: tcell.KeyCtrlA}) {
					t.Error("expected single key to be <c-a>")
				}
			})
			t.Run("<space>", func(t *testing.T) {
				keys := expectValid("<space>")
				if len(keys) != 1 {
					t.Error("expected single key")
				}
				if (keys[0] != input.Key{Key: tcell.KeyRune, Ch: ' '}) {
					t.Error("expected single key to be <space>")
				}
			})
		})

		t.Run("sequence", func(t *testing.T) {
			t.Run("characters", func(t *testing.T) {
				keys := expectValid("xyz")
				if len(keys) != 3 {
					t.Error("expected three keys")
				}
				if (keys[0] != input.Key{Key: tcell.KeyRune, Ch: 'x'}) && (keys[1] != input.Key{Key: tcell.KeyRune, Ch: 'y'}) && (keys[2] != input.Key{Key: tcell.KeyRune, Ch: 'z'}) {
					t.Error("expected sequence [x,y,z], not", keys)
				}
			})
			t.Run("with special", func(t *testing.T) {
				keys := expectValid("x<c-w>z")
				if len(keys) != 3 {
					t.Error("expected three keys")
				}
				if (keys[0] != input.Key{Key: tcell.KeyRune, Ch: 'x'}) && (keys[1] != input.Key{Key: tcell.KeyCtrlW}) && (keys[2] != input.Key{Key: tcell.KeyRune, Ch: 'z'}) {
					t.Error("expected sequence [x,<c-w>,z], not", keys)
				}
			})
		})
	})

	t.Run("valid", func(t *testing.T) {
		expectInvalid := func(s input.Keyspec) error {
			keys, err := input.ConfigKeyspecToKeys(s)
			if err == nil {
				t.Error("unexpectedly no err on invalid spec")
			}
			if keys != nil {
				t.Error("unexpected key seq on invalid spec:", keys)
			}
			return err
		}

		t.Run("unopened special", func(t *testing.T) {
			expectInvalid("c-w>")
		})
		t.Run("unclosed special (EOL)", func(t *testing.T) {
			expectInvalid("<c-w")
		})
		t.Run("unclosed special (double open)", func(t *testing.T) {
			expectInvalid("<c-w<c-a>")
		})
		t.Run("wrong delimiter in special", func(t *testing.T) {
			expectInvalid("<c+a>")
		})
	})

}

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
		emptyTree, err := input.ConstructInputTree(make(map[input.Keyspec]action.Action))
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
		tree, err := input.ConstructInputTree(map[input.Keyspec]action.Action{"xyz": &DummyAction{F: func() { shouldGetSetToTrue = true }}})
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
		tree, err := input.ConstructInputTree(map[input.Keyspec]action.Action{
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
		tree, err := input.ConstructInputTree(map[input.Keyspec]action.Action{"<asdf": &DummyAction{}})
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

func TestGetHelp(t *testing.T) {
	t.Run("Tree.GetHelp", func(t *testing.T) {
		tree, err := input.ConstructInputTree(
			map[input.Keyspec]action.Action{
				"a":  &DummyAction{S: "A"},
				"bc": &DummyAction{S: "BC"},
			},
		)
		if err != nil {
			t.Fatal("unexpectedly tree construction failed while testing help")
		}
		help := tree.GetHelp()
		if help == nil {
			t.Error("got nil help from node")
		}
		if len(help) != 2 {
			t.Error("got help with unexpected amount of entries:", len(help))
		}
		actual, ok := help["a"]
		if !ok {
			t.Error("help message for 'a' not found")
		}
		if actual != "A" {
			t.Errorf("got help string '%s' instead of expected", actual)
		}
		actual, ok = help["bc"]
		if !ok {
			t.Error("help message for 'bc' not found")
		}
		if actual != "BC" {
			t.Errorf("got help string '%s' instead of expected", actual)
		}
	})
	t.Run("Node.GetHelp", func(t *testing.T) {
		t.Run("empty node", func(t *testing.T) {
			node := input.NewNode()
			help := node.GetHelp()
			if help == nil {
				t.Error("got nil help from node")
			}
			if len(help) != 0 {
				t.Error("got non-empty help from node")
			}
		})
		t.Run("leaf", func(t *testing.T) {
			node := input.NewLeaf(&DummyAction{})
			help := node.GetHelp()
			if help == nil {
				t.Error("got nil help from leaf")
			}
			if len(help) != 1 {
				t.Error("expected one help result from leaf")
			}
		})
		t.Run("node with children", func(t *testing.T) {
			root := input.NewNode()
			lLeaf := input.NewLeaf(&DummyAction{S: "x action"})
			rMiddle := input.NewNode()
			rLeaf := input.NewLeaf(&DummyAction{S: "yz action"})

			root.Children[input.Key{Key: tcell.KeyRune, Ch: 'x'}] = lLeaf
			root.Children[input.Key{Key: tcell.KeyRune, Ch: 'y'}] = rMiddle
			rMiddle.Children[input.Key{Key: tcell.KeyRune, Ch: 'z'}] = rLeaf

			help := root.GetHelp()
			if help == nil {
				t.Error("got nil help from root")
			}
			if len(help) != 2 {
				t.Error("expected two help results from root")
			}
			actual, ok := help["x"]
			if !ok {
				t.Error("help message for 'x' not found")
			}
			if actual != "x action" {
				t.Errorf("got help string '%s' instead of expected", actual)
			}
			actual, ok = help["yz"]
			if !ok {
				t.Error("help message for 'yz' not found")
			}
			if actual != "yz action" {
				t.Errorf("got help string '%s' instead of expected", actual)
			}
		})
	})
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
	S string
}

func (d *DummyAction) Do()             { d.F() }
func (d *DummyAction) Undo()           {}
func (d *DummyAction) Undoable() bool  { return false }
func (d *DummyAction) Explain() string { return d.S }
