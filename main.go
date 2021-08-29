package main

import (
	"bufio"
	"log"
	"os"

	"dayplan/model"
	"dayplan/tui_controller"
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
  m := *model.NewModel()
	for scanner.Scan() {
		s := scanner.Text()
		m.AddEvent(*model.NewEvent(s))
	}

	tmodel := tui_model.NewTUIModel()
	tmodel.SetModel(&m)

	view := tui_view.NewTUIView(tmodel)
	defer view.Screen.Fini()

	controller := tui_controller.NewTUIController(view, tmodel)

	controller.Run()
}
