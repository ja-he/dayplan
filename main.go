package main

import (
	"bufio"
	"log"
	"os"

	"dayplan/category_style"
	"dayplan/model"
	"dayplan/tui_controller"
	"dayplan/tui_model"
	"dayplan/tui_view"
)

// MAIN
func main() {
	m := *model.NewModel()

	argc := len(os.Args)
	if argc > 1 {
		filename := os.Args[1]
		f, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot read file '%s'", filename)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			s := scanner.Text()
			m.AddEvent(*model.NewEvent(s))
		}
	}

	var catstyles category_style.CategoryStyling
	if argc > 2 {
		catstyles = *category_style.EmptyCategoryStyling()
		filename := os.Args[2]
		f, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot read file '%s'", filename)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			s := scanner.Text()
			catstyles.AddStyleFromCfg(s)
		}



	} else {
		catstyles = *category_style.DefaultCategoryStyling()
	}

	tmodel := tui_model.NewTUIModel(catstyles)
	tmodel.SetModel(&m)

	view := tui_view.NewTUIView(tmodel)
	defer view.Screen.Fini()

	controller := tui_controller.NewTUIController(view, tmodel)

	controller.Run()
}
