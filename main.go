package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"dayplan/src/category_style"
	"dayplan/src/model"
	"dayplan/src/program"
	"dayplan/src/tui"
	"dayplan/src/weather"

	"github.com/jessevdk/go-flags"
	"github.com/kelvins/sunrisesunset"
)

var owmAPIKey = os.Getenv("OWM_API_KEY")

var commandLineOpts struct {
	Day string `short:"d" long:"day" description:"Specify the day to plan" value-name:"<file>"`
}

// MAIN
func main() {
	// parse the flags
	_, err := flags.Parse(&commandLineOpts)
	if flags.WroteHelp(err) {
		os.Exit(0)
	} else if err != nil {
		panic("some flag parsing error occurred")
	}

	var programData program.Data

	// set up dir per option
	dayplanHome := os.Getenv("DAYPLAN_HOME")
	if dayplanHome == "" {
		programData.BaseDirPath = os.Getenv("HOME") + "/.config/dayplan"
	} else {
		programData.BaseDirPath = strings.TrimRight(dayplanHome, "/")
	}

	// set up day input file
	now := time.Now()
	var day model.Day

	if commandLineOpts.Day == "" {
		day = model.Day{Year: now.Year(), Month: int(now.Month()), Day: now.Day()}
	} else {
		day, err = model.FromString(commandLineOpts.Day)
		if err != nil {
			panic(err) // TODO
		}
	}

	// read category styles
	var catstyles category_style.CategoryStyling
	catstyles = *category_style.EmptyCategoryStyling()
	f, err := os.Open(programData.BaseDirPath + "/" + "category-styles")
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

	// TODO: long term I think this should be created in the controller constructor
	tmodel := tui.NewTUIModel(catstyles)

	if owmAPIKey == "" {
		tmodel.Log.Add("ERROR", "could not fetch OWM API key")
	}

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

	controller := tui.NewTUIController(view, tmodel, day, programData)

	controller.Run()
}
