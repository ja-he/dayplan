// Package editors contains the editors for the different data types.
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
	"github.com/ja-he/dayplan/internal/model"
)

// Composite implements Editor
type Composite struct {
	fields           []edit.Editor
	activeFieldIndex int
	inField          bool

	activeAndFocussedFunc func() (bool, bool)

	name         string
	quitCallback func()
}

// SwitchToNextField switches to the next field (wrapping araound, if necessary)
func (e *Composite) SwitchToNextField() {
	nextIndex := (e.activeFieldIndex + 1) % len(e.fields)
	log.Debug().Msgf("switching fields '%s' -> '%s'", e.fields[e.activeFieldIndex].GetName(), e.fields[nextIndex].GetName())
	// TODO: should _somehow_ signal deactivate to active field
	e.activeFieldIndex = nextIndex
}

// GetType asserts that this is a composite editor.
func (e *Composite) GetType() string { return "composite" }

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
func ConstructEditor[T any](name string, obj *T, extraSpec map[string]any, activeAndFocussedFunc func() (bool, bool)) (edit.Editor, error) {
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
		fields:                nil,
		activeFieldIndex:      0,
		activeAndFocussedFunc: activeAndFocussedFunc,
		name:                  name,
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
			fieldActiveAndFocussed := func() (bool, bool) {
				parentActive, parentFocussed := e.IsActiveAndFocussed()
				selfActive := parentActive && parentFocussed && e.activeFieldIndex == subeditorIndex
				return selfActive, selfActive && e.inField
			}

			// add the corresponding data to e (if not ignored)
			if !editspec.Ignore {
				switch field.Type.Kind() {
				case reflect.String:
					f := structValue.Field(i)
					e.fields = append(e.fields, &StringEditor{
						Name:              editspec.Name,
						Content:           f.String(),
						CursorPos:         0,
						ActiveAndFocussed: fieldActiveAndFocussed,
						QuitCallback: func() {
							if e.activeFieldIndex == subeditorIndex {
								e.inField = false
							}
						},
						Mode:     input.TextEditModeNormal,
						CommitFn: func(v string) { f.SetString(v) },
					})
				case reflect.Struct:

					if editspec.Ignore {
						log.Debug().Msgf("ignoring struct '%s' tagged '%s' (ignore:%t)", field.Name, editspec.Name, editspec.Ignore)
					} else {
						// construct the sub-editor for the struct
						f := structValue.Field(i)
						typedSubfield, ok := f.Addr().Interface().(*model.Category) // TODO: no clue what i was smoking here...
						if !ok {
							return nil, fmt.Errorf("unable to cast field '%s' of type '%s' to model.Category", field.Name, field.Type.String())
						}
						log.Debug().Msgf("constructing subeditor for field '%s' of type '%s'", field.Name, field.Type.String())
						sube, err := ConstructEditor(field.Name, typedSubfield, nil, fieldActiveAndFocussed)
						if err != nil {
							return nil, fmt.Errorf("unable to construct subeditor for field '%s' of type '%s' (%s)", field.Name, field.Type.String(), err.Error())
						}
						sube.AddQuitCallback(func() { e.inField = false })
						log.Debug().Msgf("successfully constructed subeditor for field '%s' of type '%s'", field.Name, field.Type.String())
						e.fields = append(e.fields, sube)
					}

				case reflect.Ptr:
					// TODO
					log.Warn().Msgf("ignoring PTR    '%s' tagged '%s' (ignore:%t) of type '%s'", field.Name, editspec.Name, editspec.Ignore, field.Type.String())
				default:
					return nil, fmt.Errorf("unable to edit non-ignored field '%s' of type '%s'", field.Name, field.Type.Kind())
				}
			}
		}

	}

	log.Debug().Msgf("have (sub?)editor with %d fields", len(e.fields))

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
	} else {
		log.Warn().Msgf("have no quit callback for editor '%s'", e.GetName())
	}
}

// CreateInputProcessor creates an input processor for the editor.
func (e *Composite) CreateInputProcessor(cfg input.InputConfig) (input.ModalInputProcessor, error) {
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

func (e *Composite) GetActiveFieldIndex() int { return e.activeFieldIndex }
func (e *Composite) IsInField() bool          { return e.inField }

func (e *Composite) IsActiveAndFocussed() (bool, bool) { return e.activeAndFocussedFunc() }

// GetFields returns the subeditors of this composite editor.
func (e *Composite) GetFields() []edit.Editor {
	return e.fields
}
