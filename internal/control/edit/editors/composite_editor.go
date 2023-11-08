package editors

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/control/action"
	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/input/processors"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/ui/panes"
)

// Composite implements Editor
type Composite struct {
	fields           []edit.Editor
	activeFieldIndex int
	inField          bool

	name         string
	quitCallback func()
}

// SwitchToNextField switches to the next field (wrapping araound, if necessary)
func (e *Composite) SwitchToNextField() {
	// TODO: should _somehow_ signal deactivate to active field
	e.activeFieldIndex = (e.activeFieldIndex + 1) % len(e.fields)
}

// SwitchToPrevField switches to the previous field (wrapping araound, if necessary)
func (e *Composite) SwitchToPrevField() {
	// TODO: should _somehow_ signal deactivate to active field
	e.activeFieldIndex = (e.activeFieldIndex - 1 + len(e.fields)) % len(e.fields)
}

// EnterField changes the editor to enter the currently selected field, e.g.
// such that input processing is deferred to the field.
func (e *Composite) EnterField() {
	if e.inField {
		log.Warn().Msgf("composite editor was prompted to enter a field despite alred being in a field; likely logic error")
	}
	e.inField = true
}

// ConstructEditor constructs a new editor...
func ConstructEditor[T any](obj *T, extraSpec map[string]any) (edit.Editor, error) {
	structPtr := reflect.ValueOf(obj)

	if structPtr.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("must pass a ptr to contruct editor (was given %s)", structPtr.Type().String())
	}
	if structPtr.IsNil() {
		return nil, fmt.Errorf("must not pass nil ptr to contruct editor")
	}
	structValue := structPtr.Elem()
	structType := structValue.Type()
	if structValue.Kind() != reflect.Struct {
		return nil, fmt.Errorf("must pass a struct (by ptr) contruct editor (was given %s (by ptr))", structType.String())
	}

	e := &Composite{
		fields:           nil,
		activeFieldIndex: 0,
		name:             "root",
	}

	// go through all tags
	for i := 0; i < structValue.NumField(); i++ {
		field := structType.Field(i)

		// when 'dpedit' set ...
		if tag, ok := field.Tag.Lookup("dpedit"); ok {
			parts := strings.Split(tag, ",")

			// build the edit spec
			editspec := dpedit{
				Name: parts[0],
			}
			if len(parts) == 2 {
				switch parts[1] {
				case "ignore":
					editspec.Ignore = true
				case "subedit": // NOTE: this is an idea, how it would be rendered is not yet imagined, might be prohibitive -> drop
					editspec.Subedit = true
				default:
					return nil, fmt.Errorf("field '%s' has unknown setting in 'dpedit': '%s'", field.Name, parts[1])
				}
			} else if len(parts) > 2 {
				return nil, fmt.Errorf("field %d has too many (%d) parts in tag 'dpedit'", i, len(parts))
			}

			subeditorIndex := i
			// add the corresponding data to e (if not ignored)
			if !editspec.Ignore {
				switch field.Type.Kind() {
				case reflect.String:
					f := structValue.Field(i)
					e.fields = append(e.fields, &StringEditor{
						Name:      editspec.Name,
						Content:   f.String(),
						CursorPos: 0,
						Active:    func() bool { return e.inField && e.activeFieldIndex == subeditorIndex },
						QuitCallback: func() {
							if e.activeFieldIndex == subeditorIndex {
								e.inField = false
							}
						},
						Mode:     input.TextEditModeNormal,
						CommitFn: func(v string) { f.SetString(v) },
					})
				case reflect.Struct:
					// TODO
					log.Warn().Msgf("ignoring STRUCT '%s' tagged '%s' (ignore:%t) of type '%s'", field.Name, editspec.Name, editspec.Ignore, field.Type.String())
				case reflect.Ptr:
					// TODO
					log.Warn().Msgf("ignoring PTR    '%s' tagged '%s' (ignore:%t) of type '%s'", field.Name, editspec.Name, editspec.Ignore, field.Type.String())
				default:
					return nil, fmt.Errorf("unable to edit non-ignored field '%s' of type '%s'", field.Name, field.Type.Kind())
				}
			}
		}

	}

	return e, nil
}

type dpedit struct {
	Name    string
	Ignore  bool
	Subedit bool
}

// GetName returns the name of the editor.
func (e *Composite) GetName() string { return e.name }

// Write writes the content of the editor back to the underlying data structure
// by calling the write functions of all subeditors.
func (e *Composite) Write() {
	for _, subeditor := range e.fields {
		subeditor.Write()
	}
}

// AddQuitCallback adds a callback that is called when the editor is quit.
func (e *Composite) AddQuitCallback(f func()) {
	if e.quitCallback != nil {
		existingCallback := e.quitCallback
		e.quitCallback = func() {
			existingCallback()
			f()
		}
	} else {
		e.quitCallback = f
	}
}

// Quit quits all subeditors and calls the quit callback.
func (e *Composite) Quit() {
	for _, subeditor := range e.fields {
		subeditor.Quit()
	}
	if e.quitCallback != nil {
		e.quitCallback()
	}
}

func (e *Composite) GetFieldCount() int {
	count := 0
	for _, subeditor := range e.fields {
		count += subeditor.GetFieldCount()
	}
	return count
}

// GetPane constructs a pane for this composite editor (including all subeditors).
func (e *Composite) GetPane(
	renderer ui.ConstrainedRenderer,
	visible func() bool,
	inputConfig input.InputConfig,
	stylesheet styling.Stylesheet,
	cursorController ui.TextCursorController,
) (ui.Pane, error) {
	subpanes := []ui.Pane{}

	rollingOffsetX := 0
	for _, subeditor := range e.fields {
		log.Debug().Msgf("constructing subpane for subeditor '%s'", subeditor.GetName())

		rollingOffsetX += 1 // padding

		subeditorOffsetX := rollingOffsetX

		subeditorH := 0
		// height is at least 1, plus 1 plus padding for any extra
		for i := 0; i < subeditor.GetFieldCount()-1; i++ {
			// TODO: this doesn't account for sub-subeditors with multiple fields
			subeditorH += 2
		}
		subeditorH += 1

		subeditorPane, err := subeditor.GetPane(
			ui.NewConstrainedRenderer(renderer, func() (int, int, int, int) {
				compositeX, compositeY, compositeW, _ := renderer.Dimensions()
				subeditorX := (compositeX + subeditorOffsetX)
				subeditorY := compositeY + 1
				subeditorW := compositeW - 2
				return subeditorX, subeditorY, subeditorW, subeditorH
			}),
			visible,
			inputConfig,
			stylesheet,
			cursorController,
		)
		if err != nil {
			return nil, fmt.Errorf("error constructing subpane for subeditor '%s' (%s)", subeditor.GetName(), err.Error())
		}
		subpanes = append(subpanes, subeditorPane)

		rollingOffsetX += subeditorH // adding space for subeditor (will be padded next)
	}
	inputProcessor, err := e.createInputProcessor(inputConfig)
	if err != nil {
		return nil, fmt.Errorf("could not construct input processor (%s)", err.Error())
	}
	return panes.NewCompositeEditorPane(
		renderer,
		visible,
		inputProcessor,
		stylesheet,
		subpanes,
		func() int { return e.activeFieldIndex },
		func() bool { return e.inField },
	), nil
}

func (e *Composite) createInputProcessor(cfg input.InputConfig) (input.ModalInputProcessor, error) {
	actionspecToFunc := map[input.Actionspec]func(){
		"next-field":      e.SwitchToNextField,
		"prev-field":      e.SwitchToPrevField,
		"enter-subeditor": e.EnterField,
		"write":           e.Write,
		"write-and-quit":  func() { e.Write(); e.Quit() },
		"quit":            e.Quit,
	}

	mappings := map[input.Keyspec]action.Action{}
	for keyspec, actionspec := range cfg.Editor {
		log.Debug().Msgf("adding mapping '%s' -> '%s'", keyspec, actionspec)
		actionspecCopy := actionspec
		mappings[keyspec] = action.NewSimple(func() string { return string(actionspecCopy) }, actionspecToFunc[actionspecCopy])
	}
	inputTree, err := input.ConstructInputTree(mappings)
	if err != nil {
		return nil, fmt.Errorf("could not construct normal mode input tree: %w", err)
	}

	return processors.NewModalInputProcessor(inputTree), nil
}
