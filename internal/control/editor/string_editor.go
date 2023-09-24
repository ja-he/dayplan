package editor

import (
	"strconv"

	"github.com/ja-he/dayplan/internal/input"
)

// A StringEditor is a control and inspect interface for editing a string.
type StringEditor interface {
	StringEditorView
	StringEditorControl
	Commit()
}

// StringEditorView allows inspection of a string editor.
type StringEditorView interface {

	// GetMode returns the current mode of the editor.
	GetMode() input.TextEditMode

	// GetCursorPos returns the current cursor position in the string, 0 being
	// the first character.
	GetCursorPos() int

	// GetContent returns the current (edited) contents.
	GetContent() string

	GetName() string

	// TODO: more
}

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

type stringEditor struct {
	Name string

	Content   string
	CursorPos int
	Mode      input.TextEditMode

	CommitFn func(string)
}

func (e stringEditor) GetName() string             { return e.Name }
func (e stringEditor) GetContent() string          { return e.Content }
func (e stringEditor) GetCursorPos() int           { return e.CursorPos }
func (e stringEditor) GetMode() input.TextEditMode { return e.Mode }

func (e *stringEditor) SetMode(m input.TextEditMode) { e.Mode = m }

func (e *stringEditor) DeleteRune() {
	tmpStr := []rune(e.Content)
	if e.CursorPos < len(tmpStr) {
		preCursor := tmpStr[:e.CursorPos]
		postCursor := tmpStr[e.CursorPos+1:]

		e.Content = string(append(preCursor, postCursor...))
	}
}

func (e *stringEditor) BackspaceRune() {
	if e.CursorPos > 0 {
		tmpStr := []rune(e.Content)
		preCursor := tmpStr[:e.CursorPos-1]
		postCursor := tmpStr[e.CursorPos:]

		e.Content = string(append(preCursor, postCursor...))
		e.CursorPos--
	}
}

func (e *stringEditor) BackspaceToBeginning() {
	nameAfterCursor := []rune(e.Content)[e.CursorPos:]
	e.Content = string(nameAfterCursor)
	e.CursorPos = 0
}

func (e *stringEditor) DeleteToEnd() {
	nameBeforeCursor := []rune(e.Content)[:e.CursorPos]
	e.Content = string(nameBeforeCursor)
}

func (e *stringEditor) Clear() {
	e.Content = ""
	e.CursorPos = 0
}

func (e *stringEditor) MoveCursorToBeginning() {
	e.CursorPos = 0
}

func (e *stringEditor) MoveCursorToEnd() {
	e.CursorPos = len([]rune(e.Content)) - 1
}

func (e *stringEditor) MoveCursorPastEnd() {
	e.CursorPos = len([]rune(e.Content))
}

func (e *stringEditor) MoveCursorLeft() {
	if e.CursorPos > 0 {
		e.CursorPos--
	}
}

func (e *stringEditor) MoveCursorRight() {
	nameLen := len([]rune(e.Content))
	if e.CursorPos+1 < nameLen {
		e.CursorPos++
	}
}

func (e *stringEditor) MoveCursorRightA() {
	nameLen := len([]rune(e.Content))
	if e.CursorPos < nameLen {
		e.CursorPos++
	}
}

func (e *stringEditor) MoveCursorNextWordBeginning() {
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

func (e *stringEditor) MoveCursorPrevWordBeginning() {
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

func (e *stringEditor) MoveCursorNextWordEnd() {
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

func (e *stringEditor) AddRune(newRune rune) {
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

func (e *stringEditor) Commit() {
	e.CommitFn(e.Content)
}
