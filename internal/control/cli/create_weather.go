package cli

import (
	"fmt"
	"strconv"

	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage"
	"github.com/ja-he/dayplan/internal/weather"
	"github.com/rs/zerolog/log"
)

func createWeatherAndSuntimes(
	envData control.EnvData,
) (*weather.Handler, storage.SunTimesProvider, error) {
	var w *weather.Handler
	var s storage.SunTimesProvider

	coordinatesProvided := (envData.Latitude != "" && envData.Longitude != "")
	owmAPIKeyProvided := (envData.OWMAPIKey != "")

	// intialize weather handler if geographic location and api key provided
	if coordinatesProvided && owmAPIKeyProvided {
		w = weather.NewHandler(envData.Latitude, envData.Longitude, envData.OWMAPIKey)
	} else {
		if !owmAPIKeyProvided {
			log.Warn().Msg("no OWM API key provided -> no weather data")
		}
		if !coordinatesProvided {
			log.Warn().Msg("no lat-/longitude provided -> no weather data")
		}
	}

	if !coordinatesProvided {
		log.Error().Msg("could not fetch lat-&longitude -> no sunrise/-set times known")
	} else {
		lat, parseErrLat := strconv.ParseFloat(envData.Latitude, 64)
		lon, parseErrLon := strconv.ParseFloat(envData.Longitude, 64)
		if parseErrLon != nil || parseErrLat != nil {
			return nil, nil, fmt.Errorf("could not parse longitude(%w)/latitude(%w)", parseErrLon, parseErrLat)
		}
		s = &model.SuntimesProvider{
			Latitude:  lat,
			Longitude: lon,
		}
	}

	return w, s, nil
}
