package main

import (
	"bufio"
	"log"
	"os"

	"dayplan/category_style"
	"dayplan/tui_controller"
	"dayplan/tui_model"
	"dayplan/tui_view"
	"dayplan/weather"
)

var owmAPIKey = os.Getenv("OWM_API_KEY")

// MAIN
func main() {
	var h *tui_controller.FileHandler

	argc := len(os.Args)
	if argc > 1 {
		filename := os.Args[1]
		h = tui_controller.NewFileHandler(filename)
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

	var owmdata []weather.OwmHourly
	if argc > 4 {
		lat := os.Args[3]
		lon := os.Args[4]
		owmdata = weather.GetHourlyInfo(lat, lon, owmAPIKey)
	}

	tmodel := tui_model.NewTUIModel(catstyles)
	tmodel.Weather = weather.GetTodaysWeather(&owmdata)

	view := tui_view.NewTUIView(tmodel)
	defer view.Screen.Fini()

	controller := tui_controller.NewTUIController(view, tmodel, h)

	controller.Run()
}
