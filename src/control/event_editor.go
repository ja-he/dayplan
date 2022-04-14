package control

import (
	"strconv"

	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/model"
)

// TODO: This could very well be its own package:
//       The editor certainly isn't TUI-specific and could be used
//       for other UIs, as long as the input is keyboard driven.

type EventEditor struct {
	Active       bool
	Original     *model.Event
	TmpEventInfo model.Event
	CursorPos    int

	Mode input.TextEditMode

	InputProcessor input.ModalInputProcessor
}

func (e *EventEditor) GetMode() input.TextEditMode  { return e.Mode }
func (e *EventEditor) SetMode(m input.TextEditMode) { e.Mode = m }

func (e *EventEditor) DeleteRune() {
	tmpStr := []rune(e.TmpEventInfo.Name)
	if e.CursorPos < len(tmpStr) {
		preCursor := tmpStr[:e.CursorPos]
		postCursor := tmpStr[e.CursorPos+1:]

		e.TmpEventInfo.Name = string(append(preCursor, postCursor...))
	}
}

func (e *EventEditor) BackspaceRune() {
	if e.CursorPos > 0 {
		tmpStr := []rune(e.TmpEventInfo.Name)
		preCursor := tmpStr[:e.CursorPos-1]
		postCursor := tmpStr[e.CursorPos:]

		e.TmpEventInfo.Name = string(append(preCursor, postCursor...))
		e.CursorPos--
	}
}

func (e *EventEditor) BackspaceToBeginning() {
	nameAfterCursor := []rune(e.TmpEventInfo.Name)[e.CursorPos:]
	e.TmpEventInfo.Name = string(nameAfterCursor)
	e.CursorPos = 0
}

func (e *EventEditor) DeleteToEnd() {
	nameBeforeCursor := []rune(e.TmpEventInfo.Name)[:e.CursorPos]
	e.TmpEventInfo.Name = string(nameBeforeCursor)
}

func (e *EventEditor) Clear() {
	e.TmpEventInfo.Name = ""
	e.CursorPos = 0
}

func (e *EventEditor) MoveCursorToBeginning() {
	e.CursorPos = 0
}

func (e *EventEditor) MoveCursorToEnd() {
	e.CursorPos = len([]rune(e.TmpEventInfo.Name)) - 1
}

func (e *EventEditor) MoveCursorPastEnd() {
	e.CursorPos = len([]rune(e.TmpEventInfo.Name))
}

func (e *EventEditor) MoveCursorLeft() {
	if e.CursorPos > 0 {
		e.CursorPos--
	}
}

func (e *EventEditor) MoveCursorRight() {
	nameLen := len([]rune(e.TmpEventInfo.Name))
	if e.CursorPos+1 < nameLen {
		e.CursorPos++
	}
}

func (e *EventEditor) MoveCursorRightA() {
	nameLen := len([]rune(e.TmpEventInfo.Name))
	if e.CursorPos < nameLen {
		e.CursorPos++
	}
}

func (e *EventEditor) MoveCursorNextWordBeginning() {
	nameAfterCursor := []rune(e.TmpEventInfo.Name)[e.CursorPos:]
	i := 0
	for i < len(nameAfterCursor) && nameAfterCursor[i] != ' ' {
		i++
	}
	for i < len(nameAfterCursor) && nameAfterCursor[i] == ' ' {
		i++
	}
	newCursorPos := e.CursorPos + i
	if newCursorPos < len([]rune(e.TmpEventInfo.Name)) {
		e.CursorPos = newCursorPos
	} else {
		e.MoveCursorToEnd()
	}
}

func (e *EventEditor) MoveCursorPrevWordBeginning() {
	nameBeforeCursor := []rune(e.TmpEventInfo.Name)[:e.CursorPos]
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

func (e *EventEditor) MoveCursorNextWordEnd() {
	nameAfterCursor := []rune(e.TmpEventInfo.Name)[e.CursorPos:]
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
	if newCursorPos < len([]rune(e.TmpEventInfo.Name)) {
		e.CursorPos = newCursorPos
	} else {
		e.MoveCursorToEnd()
	}
}

func (e *EventEditor) AddRune(newRune rune) {
	if strconv.IsPrint(newRune) {
		tmpName := []rune(e.TmpEventInfo.Name)
		cursorPos := e.CursorPos
		if len(tmpName) == cursorPos {
			tmpName = append(tmpName, newRune)
		} else {
			tmpName = append(tmpName[:cursorPos+1], tmpName[cursorPos:]...)
			tmpName[cursorPos] = newRune
		}
		e.TmpEventInfo.Name = string(tmpName)
		e.CursorPos++
	}
}
