package editors

import (
	"fmt"
	"strconv"

	"github.com/ja-he/dayplan/internal/control/action"
	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/input/processors"
	"github.com/rs/zerolog/log"
)

// StringEditorControl allows manipulation of a string editor.
type StringEditorControl interface {
	SetMode(m input.TextEditMode)
	DeleteRune()
	BackspaceRune()
	BackspaceToBeginning()
	DeleteToEnd()
	Clear()
	MoveCursorToBeginning()
	MoveCursorToEnd()
	MoveCursorPastEnd()
	MoveCursorLeft()
	MoveCursorRight()
	MoveCursorRightA()
	MoveCursorNextWordBeginning()
	MoveCursorPrevWordBeginning()
	MoveCursorNextWordEnd()
	AddRune(newRune rune)
}

// StringEditor ...
type StringEditor struct {
	ID string

	Content   string
	CursorPos int
	Mode      input.TextEditMode
	StatusFn  func() edit.EditorStatus

	parent *Composite

	QuitCallback func()

	CommitFn func(string)
}

// GetType asserts that this is a string editor.
func (e *StringEditor) GetType() string { return "string" }

// GetStatus ...
func (e StringEditor) GetStatus() edit.EditorStatus {
	if e.parent == nil {
		return edit.EditorFocussed
	}

	parentEditorStatus := e.parent.GetStatus()
	switch parentEditorStatus {
	case edit.EditorInactive, edit.EditorSelected:
		return edit.EditorInactive
	case edit.EditorDescendantActive, edit.EditorFocussed:
		if e.parent.activeFieldID == e.ID {
			if e.parent.inField {
				return edit.EditorFocussed
			}
			return edit.EditorSelected
		}
		return edit.EditorInactive
	default:
		log.Error().Msgf("invalid edit state found (%s) likely logic error", parentEditorStatus)
		return edit.EditorInactive
	}
}

// GetID returns the ID of the editor.
func (e StringEditor) GetID() string { return e.ID }

// GetContent returns the current (edited) contents.
func (e StringEditor) GetContent() string { return e.Content }

// GetCursorPos returns the current cursor position in the string, 0 being
func (e StringEditor) GetCursorPos() int { return e.CursorPos }

// GetMode returns the current mode of the editor.
func (e StringEditor) GetMode() input.TextEditMode { return e.Mode }

// SetMode sets the current mode of the editor.
func (e *StringEditor) SetMode(m input.TextEditMode) { e.Mode = m }

// DeleteRune deletes the rune at the cursor position.
func (e *StringEditor) DeleteRune() {
	tmpStr := []rune(e.Content)
	if e.CursorPos < len(tmpStr) {
		preCursor := tmpStr[:e.CursorPos]
		postCursor := tmpStr[e.CursorPos+1:]

		e.Content = string(append(preCursor, postCursor...))
	}
}

// BackspaceRune deletes the rune before the cursor position.
func (e *StringEditor) BackspaceRune() {
	if e.CursorPos > 0 {
		tmpStr := []rune(e.Content)
		preCursor := tmpStr[:e.CursorPos-1]
		postCursor := tmpStr[e.CursorPos:]

		e.Content = string(append(preCursor, postCursor...))
		e.CursorPos--
	}
}

// BackspaceToBeginning deletes all runes before the cursor position.
func (e *StringEditor) BackspaceToBeginning() {
	nameAfterCursor := []rune(e.Content)[e.CursorPos:]
	e.Content = string(nameAfterCursor)
	e.CursorPos = 0
}

// DeleteToEnd deletes all runes after the cursor position.
func (e *StringEditor) DeleteToEnd() {
	nameBeforeCursor := []rune(e.Content)[:e.CursorPos]
	e.Content = string(nameBeforeCursor)
}

// Clear deletes all runes in the editor.
func (e *StringEditor) Clear() {
	e.Content = ""
	e.CursorPos = 0
}

// MoveCursorToBeginning moves the cursor to the beginning of the string.
func (e *StringEditor) MoveCursorToBeginning() {
	e.CursorPos = 0
}

// MoveCursorToEnd moves the cursor to the end of the string.
func (e *StringEditor) MoveCursorToEnd() {
	e.CursorPos = len([]rune(e.Content)) - 1
}

// MoveCursorPastEnd moves the cursor past the end of the string.
func (e *StringEditor) MoveCursorPastEnd() {
	e.CursorPos = len([]rune(e.Content))
}

// MoveCursorLeft moves the cursor one rune to the left.
func (e *StringEditor) MoveCursorLeft() {
	if e.CursorPos > 0 {
		e.CursorPos--
	}
}

// MoveCursorRight moves the cursor one rune to the right.
func (e *StringEditor) MoveCursorRight() {
	nameLen := len([]rune(e.Content))
	if e.CursorPos+1 < nameLen {
		e.CursorPos++
	}
}

// MoveCursorNextWordBeginning moves the cursor one rune to the right, or to
// the end of the string if already at the end.
func (e *StringEditor) MoveCursorNextWordBeginning() {
	if len([]rune(e.Content)) == 0 {
		e.CursorPos = 0
		return
	}

	nameAfterCursor := []rune(e.Content)[e.CursorPos:]
	i := 0
	for i < len(nameAfterCursor) && nameAfterCursor[i] != ' ' {
		i++
	}
	for i < len(nameAfterCursor) && nameAfterCursor[i] == ' ' {
		i++
	}
	newCursorPos := e.CursorPos + i
	if newCursorPos < len([]rune(e.Content)) {
		e.CursorPos = newCursorPos
	} else {
		e.MoveCursorToEnd()
	}
}

// MoveCursorPrevWordBeginning moves the cursor one rune to the left, or to the
// beginning of the string if already at the beginning.
func (e *StringEditor) MoveCursorPrevWordBeginning() {
	nameBeforeCursor := []rune(e.Content)[:e.CursorPos]
	if len(nameBeforeCursor) == 0 {
		return
	}
	i := len(nameBeforeCursor) - 1
	for i > 0 && nameBeforeCursor[i-1] == ' ' {
		i--
	}
	for i > 0 && nameBeforeCursor[i-1] != ' ' {
		i--
	}
	e.CursorPos = i
}

// MoveCursorNextWordEnd moves the cursor to the end of the next word.
func (e *StringEditor) MoveCursorNextWordEnd() {
	nameAfterCursor := []rune(e.Content)[e.CursorPos:]
	if len(nameAfterCursor) == 0 {
		return
	}

	i := 0
	for i < len(nameAfterCursor)-1 && nameAfterCursor[i+1] == ' ' {
		i++
	}
	for i < len(nameAfterCursor)-1 && nameAfterCursor[i+1] != ' ' {
		i++
	}
	newCursorPos := e.CursorPos + i
	if newCursorPos < len([]rune(e.Content)) {
		e.CursorPos = newCursorPos
	} else {
		e.MoveCursorToEnd()
	}
}

// AddRune adds a rune at the cursor position.
func (e *StringEditor) AddRune(newRune rune) {
	if strconv.IsPrint(newRune) {
		tmpName := []rune(e.Content)
		cursorPos := e.CursorPos
		if len(tmpName) == cursorPos {
			tmpName = append(tmpName, newRune)
		} else {
			tmpName = append(tmpName[:cursorPos+1], tmpName[cursorPos:]...)
			tmpName[cursorPos] = newRune
		}
		e.Content = string(tmpName)
		e.CursorPos++
	}
}

func (e *StringEditor) GetFieldCount() int {
	return 1
}

// Write commits the current contents of the editor.
func (e *StringEditor) Write() {
	e.CommitFn(e.Content)
}

// Quit the editor.
func (e *StringEditor) Quit() {
	e.QuitCallback()
}

// AddQuitCallback adds a callback that is called when the editor is quit.
func (e *StringEditor) AddQuitCallback(f func()) {
	if e.QuitCallback != nil {
		existingCallback := e.QuitCallback
		e.QuitCallback = func() {
			existingCallback()
			f()
		}
	}
}

// CreateInputProcessor creates an input processor for the editor.
func (e *StringEditor) CreateInputProcessor(cfg input.InputConfig) (input.ModalInputProcessor, error) {

	var enterInsertMode func()
	var exitInsertMode func()

	actionspecToFunc := map[input.Actionspec]func(){
		"move-cursor-rune-left":    e.MoveCursorLeft,
		"move-cursor-rune-right":   e.MoveCursorRight,
		"move-cursor-to-beginning": e.MoveCursorToBeginning,
		"move-cursor-to-end":       e.MoveCursorToEnd,
		"write":                    e.Write,
		"quit":                     e.Quit,
		"backspace":                e.BackspaceRune,
		"backspace-to-beginning":   e.BackspaceToBeginning,
		"delete-rune":              e.DeleteRune,
		"delete-rune-and-insert":   func() { e.DeleteRune(); enterInsertMode() },
		"delete-to-end":            e.DeleteToEnd,
		"delete-to-end-and-insert": func() { e.DeleteToEnd(); enterInsertMode() },
		"swap-mode-insert":         func() { enterInsertMode() },
		"swap-mode-normal":         func() { exitInsertMode() },
	}

	normalModeMappings := map[input.Keyspec]action.Action{}
	for keyspec, actionspec := range cfg.StringEditor.Normal {
		actionspecCopy := actionspec
		normalModeMappings[keyspec] = action.NewSimple(func() string { return string(actionspecCopy) }, actionspecToFunc[actionspecCopy])
	}
	normalModeInputTree, err := input.ConstructInputTree(normalModeMappings)
	if err != nil {
		return nil, fmt.Errorf("could not construct normal mode input tree: %w", err)
	}

	insertModeMappings := map[input.Keyspec]action.Action{}
	for keyspec, actionspec := range cfg.StringEditor.Insert {
		actionspecCopy := actionspec
		insertModeMappings[keyspec] = action.NewSimple(func() string { return string(actionspecCopy) }, actionspecToFunc[actionspecCopy])
	}
	insertModeInputTree, err := processors.NewTextInputProcessor(insertModeMappings, e.AddRune)
	if err != nil {
		return nil, fmt.Errorf("could not construct insert mode input processor: %w", err)
	}

	p := processors.NewModalInputProcessor(normalModeInputTree)
	enterInsertMode = func() {
		log.Debug().Msgf("entering insert mode")
		p.ApplyModalOverlay(insertModeInputTree)
		e.SetMode(input.TextEditModeInsert)
	}
	exitInsertMode = func() {
		log.Debug().Msgf("entering normal mode")
		p.PopModalOverlay()
		e.SetMode(input.TextEditModeNormal)
	}
	log.Debug().Msgf("attached mode swapping functions")

	return p, nil
}
