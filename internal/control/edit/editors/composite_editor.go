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
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/ui/panes"
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
						typedSubfield, ok := f.Addr().Interface().(*model.Category)
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

// GetPane constructs a pane for this composite editor (including all subeditors).
func (e *Composite) GetPane(
	renderer ui.ConstrainedRenderer,
	visible func() bool,
	inputConfig input.InputConfig,
	stylesheet styling.Stylesheet,
	cursorController ui.CursorLocationRequestHandler,
) (ui.Pane, error) {
	subpanes := []ui.Pane{}

	// TODO: this needs to compute an enriched version of the editor tree
	editorSummary := e.GetSummary()
	minX, minY, maxWidth, maxHeight := renderer.Dimensions()
	uiBoxModel, err := translateToUIBoxModel(editorSummary, minX, minY, maxWidth, maxHeight)
	if err != nil {
		return nil, fmt.Errorf("error translating editor summary to UI box model (%s)", err.Error())
	}
	log.Debug().Msgf("have UI box model: %s", uiBoxModel.String())

	for _, child := range uiBoxModel.Children {
		childX, childY, childW, childH := child.X, child.Y, child.W, child.H
		subRenderer := ui.NewConstrainedRenderer(renderer, func() (int, int, int, int) { return childX, childY, childW, childH })
		subeditorPane, err := child.Represents.GetPane(
			subRenderer,
			visible,
			inputConfig,
			stylesheet,
			cursorController,
		)
		if err != nil {
			return nil, fmt.Errorf("error constructing subpane of '%s' for subeditor '%s' (%s)", e.name, child.Represents.GetName(), err.Error())
		}
		subpanes = append(subpanes, subeditorPane)
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
		e,
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

func (e *Composite) IsActiveAndFocussed() (bool, bool) { return e.activeAndFocussedFunc() }

func (e *Composite) GetSummary() edit.SummaryEntry {

	result := edit.SummaryEntry{
		Representation: []edit.SummaryEntry{},
		Represents:     e,
	}
	for _, subeditor := range e.fields {
		log.Debug().Msgf("constructing subpane of '%s' for subeditor '%s'", e.name, subeditor.GetName())
		result.Representation = append(result.Representation.([]edit.SummaryEntry), subeditor.GetSummary())
	}

	return result
}

func translateToUIBoxModel(summary edit.SummaryEntry, minX, minY, maxWidth, maxHeight int) (ui.BoxRepresentation[edit.Editor], error) {

	switch repr := summary.Representation.(type) {

	// a slice indicates a composite
	case []edit.SummaryEntry:
		var children []ui.BoxRepresentation[edit.Editor]
		computedHeight := 1
		rollingY := minY + 1
		for _, child := range repr {
			childBoxRepresentation, err := translateToUIBoxModel(child, minX+1, rollingY, maxWidth-2, maxHeight-2)
			if err != nil {
				return ui.BoxRepresentation[edit.Editor]{}, fmt.Errorf("error translating child '%s' (%s)", child.Represents.GetName(), err.Error())
			}
			rollingY += childBoxRepresentation.H + 1
			children = append(children, childBoxRepresentation)
			computedHeight += childBoxRepresentation.H + 1
		}
		return ui.BoxRepresentation[edit.Editor]{
			X:          minX,
			Y:          minY,
			W:          maxWidth,
			H:          computedHeight,
			Represents: summary.Represents,
			Children:   children,
		}, nil

	// a string indicates a leaf, i.e., a concrete editor rather than a composite
	case string:
		switch repr {
		case "string":
			return ui.BoxRepresentation[edit.Editor]{
				X:          minX,
				Y:          minY,
				W:          maxWidth,
				H:          1,
				Represents: summary.Represents,
				Children:   nil,
			}, nil
		default:
			return ui.BoxRepresentation[edit.Editor]{}, fmt.Errorf("unknown editor identification value '%s'", repr)
		}

	default:
		return ui.BoxRepresentation[edit.Editor]{}, fmt.Errorf("for editor '%s' have unknown type '%t'", summary.Represents.GetName(), summary.Representation)

	}

}
