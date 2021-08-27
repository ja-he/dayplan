package main

import (
	"bufio"
	"log"
	"os"

	"dayplan/model"
	"dayplan/tui_model"
	"dayplan/tui_view"
)

// MAIN
func main() {
	filename := os.Args[1]
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot read file '%s'", filename)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var m model.Model
	for scanner.Scan() {
		s := scanner.Text()
		m.AddEvent(*model.NewEvent(s))
	}

	t := tui_model.NewTUIModel()
	t.SetModel(&m)

	view := tui_view.NewTUIView(t)
	defer view.Screen.Fini()

	view.Run()
}
