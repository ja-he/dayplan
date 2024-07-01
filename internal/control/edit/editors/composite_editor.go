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
)

// An EditorID is a unique identifier for an editor within a composite editor.
type EditorID = string

// Composite implements Editor
type Composite struct {
	fields        map[EditorID]edit.Editor
	activeFieldID EditorID
	fieldOrder    []EditorID
	inField       bool

	parent *Composite

	id           EditorID
	quitCallback func()
}

func (e *Composite) getCurrentFieldIndex() int {
	for i, id := range e.fieldOrder {
		if id == e.activeFieldID {
			return i
		}
	}
	log.Warn().Msg("could not find a composite editor field index (will provide 0)")
	return 0
}

// SwitchToNextField switches to the next field (wrapping araound, if necessary)
func (e *Composite) SwitchToNextField() {
	log.Trace().Interface("fieldOrder", e.fieldOrder).Msgf("switching to next field")
	prevID := e.activeFieldID
	indexOfCurrent := e.getCurrentFieldIndex()
	nextIndex := (indexOfCurrent + 1) % len(e.fieldOrder)
	nextID := e.fieldOrder[nextIndex]
	log.Debug().Msgf("switching fields '%s' -> '%s'", e.fields[prevID].GetID(), e.fields[nextID].GetID())
	// TODO: should _somehow_ signal deactivate to active field (or perhaps not, not necessary in the current design imo)
	e.activeFieldID = e.fieldOrder[nextIndex]
}

// GetType asserts that this is a composite editor.
func (e *Composite) GetType() string { return "composite" }

// SwitchToPrevField switches to the previous field (wrapping araound, if necessary)
func (e *Composite) SwitchToPrevField() {
	prevID := e.activeFieldID
	indexOfCurrent := e.getCurrentFieldIndex()
	nextIndex := (indexOfCurrent - 1 + len(e.fieldOrder)) % len(e.fieldOrder)
	log.Debug().Msgf("switching fields '%s' -> '%s'", e.fields[prevID].GetID(), e.fields[e.fieldOrder[nextIndex]].GetID())
	e.activeFieldID = e.fieldOrder[nextIndex]
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
func ConstructEditor(id string, obj any, extraSpec map[string]any, parentEditor *Composite) (edit.Editor, error) {
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

	constructedCompositeEditor := &Composite{
		fields:        make(map[EditorID]edit.Editor),
		activeFieldID: "___unassigned", // NOTE: this must be done in the following
		fieldOrder:    nil,             // NOTE: this must be done in the following
		id:            id,
		parent:        parentEditor,
	}

	// go through all tags
	for i := 0; i < structValue.NumField(); i++ {
		field := structType.Field(i)

		// when 'dpedit' set ...
		if tag, ok := field.Tag.Lookup("dpedit"); ok {
			parts := strings.Split(tag, ",")

			// build the edit spec
			editspec := dpedit{
				ID: parts[0],
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

			// add the corresponding data to e (if not ignored)
			if !editspec.Ignore {
				// set the active field to the first field
				if constructedCompositeEditor.activeFieldID == "___unassigned" {
					constructedCompositeEditor.activeFieldID = editspec.ID
				}
				constructedCompositeEditor.fieldOrder = append(constructedCompositeEditor.fieldOrder, editspec.ID)

				switch field.Type.Kind() {
				case reflect.String:
					f := structValue.Field(i)
					constructedCompositeEditor.fields[editspec.ID] = &StringEditor{
						ID:        editspec.ID,
						Content:   f.String(),
						CursorPos: 0,
						QuitCallback: func() {
							if constructedCompositeEditor.activeFieldID == editspec.ID {
								constructedCompositeEditor.inField = false
							}
						},
						Mode:     input.TextEditModeNormal,
						CommitFn: func(v string) { f.SetString(v) },
						parent:   constructedCompositeEditor,
					}
				case reflect.Struct:

					if editspec.Ignore {
						log.Debug().Msgf("ignoring struct '%s' tagged '%s' (ignore:%t)", field.Name, editspec.ID, editspec.Ignore)
					} else {
						// construct the sub-editor for the struct
						f := structValue.Field(i)
						var fAsPtr any
						if f.Kind() == reflect.Ptr {
							fAsPtr = f.Interface()
						} else {
							fAsPtr = f.Addr().Interface()
						}
						log.Debug().Msgf("constructing subeditor for field '%s' (tagged '%s') of type '%s'", field.Name, editspec.ID, field.Type.String())
						sube, err := ConstructEditor(editspec.ID, fAsPtr, nil, constructedCompositeEditor)
						if err != nil {
							return nil, fmt.Errorf("unable to construct subeditor for field '%s' (tagged '%s') of type '%s' (%s)", field.Name, editspec.ID, field.Type.String(), err.Error())
						}
						sube.AddQuitCallback(func() { constructedCompositeEditor.inField = false })
						log.Debug().Msgf("successfully constructed subeditor for field '%s' (tagged '%s') of type '%s'", field.Name, editspec.ID, field.Type.String())
						constructedCompositeEditor.fields[editspec.ID] = sube
					}

				case reflect.Ptr:
					// TODO
					log.Warn().Msgf("ignoring PTR    '%s' tagged '%s' (ignore:%t) of type '%s'", field.Name, editspec.ID, editspec.Ignore, field.Type.String())
				default:
					return nil, fmt.Errorf("unable to edit non-ignored field '%s' (tagged '%s') of type '%s'", field.Name, editspec.ID, field.Type.Kind())
				}
			}
		}

	}

	if len(constructedCompositeEditor.fieldOrder) == 0 {
		return nil, fmt.Errorf("could not find any fields to edit")
	}
	if constructedCompositeEditor.activeFieldID == "___unassigned" {
		return nil, fmt.Errorf("could not find a field to set as active")
	}

	log.Debug().Msgf("have (sub?)editor with %d fields", len(constructedCompositeEditor.fields))

	return constructedCompositeEditor, nil
}

// GetStatus informs on whether the editor is active and focussed.
//
// "active" here means that the editor is in use, i.e. the user is currently
// editing within the editor.
// "focussed" means that the editor is the one currently receiving input,
// i.e. that it is the "lowestmost" active editor.
//
// E.g. when there is merely a single string editor, it must be active and
// focused.
// E.g. when there is a composite editor it must be active but the focus may
// lie with it or with a child editor.
func (e *Composite) GetStatus() edit.EditorStatus {
	parentEditor := e.parent
	// if there is no parent editor we are the root, ergo we can assume to have focus
	if parentEditor == nil {
		if e.inField {
			return edit.EditorDescendantActive
		}
		return edit.EditorFocussed
	}
	parentStatus := parentEditor.GetStatus()
	switch parentStatus {
	case edit.EditorInactive, edit.EditorSelected:
		return edit.EditorInactive
	case edit.EditorDescendantActive, edit.EditorFocussed:
		if parentEditor.activeFieldID == e.id {
			if parentEditor.inField {
				if e.inField {
					return edit.EditorDescendantActive
				}
				return edit.EditorFocussed
			}
			return edit.EditorSelected
		}
		return edit.EditorInactive
	default:
		log.Error().Msgf("invalid edit state found (%s) likely logic error", parentStatus)
		return edit.EditorInactive
	}
}

type dpedit struct {
	ID      string
	Ignore  bool
	Subedit bool
}

// GetID returns the ID of the editor.
func (e *Composite) GetID() string { return e.id }

// GetFieldOrder returns the order of the fields.
func (e *Composite) GetFieldOrder() []EditorID { return e.fieldOrder }

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
		log.Warn().Msgf("have no quit callback for editor '%s'", e.GetID())
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

// GetActiveFieldID returns the ID of the currently active field.
func (e *Composite) GetActiveFieldID() EditorID { return e.activeFieldID }

// IsInField informs on whether the editor is currently in a field.
func (e *Composite) IsInField() bool { return e.inField }

// GetFields returns the subeditors of this composite editor.
//
// TOOD: should this exist / be public (what is it good for)?
func (e *Composite) GetFields() map[EditorID]edit.Editor {
	return e.fields
}
