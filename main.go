package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gdamore/tcell/v2"
)

func initScreen() tcell.Screen {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
  return s
}

func main() {
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)

	// Initialize screen
  s := initScreen()
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.EnablePaste()
	s.Clear()

	for {
		// Update screen
		s.Show()
		s.Clear()

		// Poll event
		ev := s.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			// TODO: handle keys
		case *tcell.EventMouse:
			// TODO: handle mouse input
		}
	}
}
