package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"dayplan/src/category_style"
	"dayplan/src/model"
	"dayplan/src/tui"
	"dayplan/src/weather"

	"github.com/jessevdk/go-flags"
	"github.com/kelvins/sunrisesunset"
)

var owmAPIKey = os.Getenv("OWM_API_KEY")

var opts struct {
	Dir string `short:"c" long:"config-dir" description:"Specify the directory dayplan uses" value-name:"<dir>"`
	Day string `short:"d" long:"day" description:"Specify the day to plan" value-name:"<file>"`
}

// MAIN
func main() {
	// parse the flags
	_, err := flags.Parse(&opts)
	if flags.WroteHelp(err) {
		os.Exit(0)
	} else if err != nil {
		panic("some flag parsing error occurred")
	}

	// set up dir per option
	var dir string
	if opts.Dir == "" {
		dir = os.Getenv("HOME") + "/.config/dayplan"
	} else {
		dir = strings.TrimRight(opts.Dir, "/")
	}

	// set up day input file
	now := time.Now()
	var day string
	if opts.Day == "" {
		day = fmt.Sprintf("%04d-%02d-%02d", now.Year(), now.Month(), now.Day())
	} else {
		day = opts.Day
	}
	dayFile := tui.NewFileHandler(dir + "/days/" + day)

	// read category styles
	var catstyles category_style.CategoryStyling
	catstyles = *category_style.EmptyCategoryStyling()
	f, err := os.Open(dir + "/" + "category-styles")
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			s := scanner.Text()
			catstyles.AddStyleFromCfg(s)
		}
		f.Close()
	} else {
		catstyles = *category_style.DefaultCategoryStyling()
	}

	tmodel := tui.NewTUIModel(catstyles)

	lat := os.Getenv("LATITUDE")
	lon := os.Getenv("LONGITUDE")
	coordinatesProvided := (lat != "" && lon != "")
	if coordinatesProvided {
		tmodel.Weather = *weather.NewHandler(lat, lon, owmAPIKey)

		latF, _ := strconv.ParseFloat(lat, 64)
		lonF, _ := strconv.ParseFloat(lon, 64)
		_, utcDeltaSeconds := time.Now().Zone()
		utcDeltaHours := utcDeltaSeconds / (60 * 60)
		sunrise, sunset, err := sunrisesunset.GetSunriseSunset(latF, lonF,
			float64(utcDeltaHours), time.Now())
		if err != nil {
			log.Fatalf("error getting sunrise/-set '%s'", err)
		}
		tmodel.SunTimes.Rise = *model.NewTimestampFromGotime(sunrise)
		tmodel.SunTimes.Set = *model.NewTimestampFromGotime(sunset)
	} else {
		tmodel.SunTimes.Rise = *model.NewTimestamp("00:00")
		tmodel.SunTimes.Set = *model.NewTimestamp("23:59")
		// TODO: we should differentiate status and error, and probably have a more
		//       robust error logging system, that users can view while the program
		//       is running.
		tmodel.Log.Add("ERROR", "could not fetch lat-&longitude")
	}

	view := tui.NewTUIView(tmodel)
	defer view.Screen.Fini()

	controller := tui.NewTUIController(view, tmodel, dayFile)

	controller.Run()
}
