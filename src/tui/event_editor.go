package tui

import (
	"dayplan/src/model"
	"strconv"
)

// TODO: This could very well be its own package:
//       The editor certainly isn't TUI-specific and could be used
//       for other UIs, as long as the input is keyboard driven.

type EventEditor struct {
	Active       bool
	TmpEventInfo model.Event
	CursorPos    int
}

func (e *EventEditor) backspaceChar() {
	if e.CursorPos > 0 {
		tmpStr := []rune(e.TmpEventInfo.Name)
		preCursor := tmpStr[:e.CursorPos-1]
		postCursor := tmpStr[e.CursorPos:]

		e.TmpEventInfo.Name = string(append(preCursor, postCursor...))
		e.CursorPos--
	}
}

func (e *EventEditor) deleteToBeginning() {
	nameAfterCursor := []rune(e.TmpEventInfo.Name)[e.CursorPos:]
	e.TmpEventInfo.Name = string(nameAfterCursor)
	e.CursorPos = 0
}

func (e *EventEditor) moveCursorToBeginning() {
	e.CursorPos = 0
}

func (e *EventEditor) moveCursorToEnd() {
	e.CursorPos = len([]rune(e.TmpEventInfo.Name))
}

func (e *EventEditor) moveCursorLeft() {
	if e.CursorPos > 0 {
		e.CursorPos--
	}
}

func (e *EventEditor) moveCursorRight() {
	nameLen := len([]rune(e.TmpEventInfo.Name))
	if e.CursorPos < nameLen {
		e.CursorPos++
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
