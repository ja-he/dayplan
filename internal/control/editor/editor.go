// Package editor implements editing for the various types.
// Besides some legacy editing, the main editor implemented is a generic editor E.
// It implements the Editor interface, including the inspect-only View interface.
package editor

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/input"
)

// View is the inspect interface of an editor.
type View interface {
	GetViews() []StringEditorView    // TODO: there must be a way to get other editor thingys
	GetView(string) StringEditorView // TODO: there must be a way to get other editor thingys
}

// Editor is the control and inspect interface of an editor.
type Editor interface {
	View

	GetEditor(string) StringEditor // TODO: there must be a way to get other editor thingys
	SwitchToNextField()
	SwitchToPrevField()
}

// E implements Editor
type E struct {
	fields           []*stringEditor
	activeFieldIndex int
}

// TODO:
//   anywhere where StringEditorView is used, it should be replaced by a more
//   general thing which I have not yet come up with.

// GetViews...
func (e *E) GetViews() []StringEditorView {
	fields := make([]StringEditorView, len(e.fields))
	for i, f := range e.fields {
		fields[i] = f
	}
	return fields
}

// GetView returns a view by its name.
func (e *E) GetView(name string) StringEditorView {
	for _, f := range e.fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// GetEditor returns an editor by its name.
func (e *E) GetEditor(name string) StringEditor {
	for _, f := range e.fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// SwitchToNextField switches to the next field (wrapping araound, if necessary)
func (e *E) SwitchToNextField() {
	// TODO: should _somehow_ signal deactivate to active field
	e.activeFieldIndex = (e.activeFieldIndex + 1) % len(e.fields)
}

// SwitchToPrevField switches to the previous field (wrapping araound, if necessary)
func (e *E) SwitchToPrevField() {
	// TODO: should _somehow_ signal deactivate to active field
	e.activeFieldIndex = (e.activeFieldIndex - 1 + len(e.fields)) % len(e.fields)
}

// ConstructEditor constructs a new editor...
func ConstructEditor[T any](obj *T, extraSpec map[string]any) (Editor, error) {
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

	e := &E{
		fields:           nil,
		activeFieldIndex: 0,
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

			// add the corresponding data to e (if not ignored)
			if !editspec.Ignore {
				switch field.Type.Kind() {
				case reflect.String:
					f := structValue.Field(i)
					e.fields = append(e.fields, &stringEditor{
						Name:      editspec.Name,
						Content:   f.String(),
						CursorPos: 0,
						Mode:      input.TextEditModeNormal,
						CommitFn:  func(v string) { f.SetString(v) },
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
