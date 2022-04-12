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

	inputProcessor input.ModalInputProcessor
}

func (e *EventEditor) deleteRune() {
	tmpStr := []rune(e.TmpEventInfo.Name)
	if e.CursorPos < len(tmpStr) {
		preCursor := tmpStr[:e.CursorPos]
		postCursor := tmpStr[e.CursorPos+1:]

		e.TmpEventInfo.Name = string(append(preCursor, postCursor...))
	}
}

func (e *EventEditor) backspaceRune() {
	if e.CursorPos > 0 {
		tmpStr := []rune(e.TmpEventInfo.Name)
		preCursor := tmpStr[:e.CursorPos-1]
		postCursor := tmpStr[e.CursorPos:]

		e.TmpEventInfo.Name = string(append(preCursor, postCursor...))
		e.CursorPos--
	}
}

func (e *EventEditor) backspaceToBeginning() {
	nameAfterCursor := []rune(e.TmpEventInfo.Name)[e.CursorPos:]
	e.TmpEventInfo.Name = string(nameAfterCursor)
	e.CursorPos = 0
}

func (e *EventEditor) deleteToEnd() {
	nameBeforeCursor := []rune(e.TmpEventInfo.Name)[:e.CursorPos]
	e.TmpEventInfo.Name = string(nameBeforeCursor)
}

func (e *EventEditor) clear() {
	e.TmpEventInfo.Name = ""
	e.CursorPos = 0
}

func (e *EventEditor) moveCursorToBeginning() {
	e.CursorPos = 0
}

func (e *EventEditor) moveCursorToEnd() {
	e.CursorPos = len([]rune(e.TmpEventInfo.Name)) - 1
}

func (e *EventEditor) moveCursorPastEnd() {
	e.CursorPos = len([]rune(e.TmpEventInfo.Name))
}

func (e *EventEditor) moveCursorLeft() {
	if e.CursorPos > 0 {
		e.CursorPos--
	}
}

func (e *EventEditor) moveCursorRight() {
	nameLen := len([]rune(e.TmpEventInfo.Name))
	if e.CursorPos+1 < nameLen {
		e.CursorPos++
	}
}

func (e *EventEditor) moveCursorRightA() {
	nameLen := len([]rune(e.TmpEventInfo.Name))
	if e.CursorPos < nameLen {
		e.CursorPos++
	}
}

func (e *EventEditor) moveCursorNextWordBeginning() {
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
		e.moveCursorToEnd()
	}
}

func (e *EventEditor) moveCursorPrevWordBeginning() {
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

func (e *EventEditor) moveCursorNextWordEnd() {
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
		e.moveCursorToEnd()
	}
}

func (e *EventEditor) addRune(newRune rune) {
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
