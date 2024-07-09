// Package weather provides a handler for fetching weather data from OpenWeatherMap.
package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ja-he/dayplan/internal/model"
)

// OWMWeather represents the weather data from OpenWeatherMap.
type OWMWeather struct {
	ID          int    `json:"id"`          // 801,
	Main        string `json:"main"`        // "Clouds",
	Description string `json:"description"` // "few clouds",
	Icon        string `json:"icon"`        // "02d"
}

// OWMHourly represents the hourly weather data from OpenWeatherMap.
type OWMHourly struct {
	Dt         uint64       `json:"dt"`         // 1630429200,
	Temp       float64      `json:"temp"`       // 290.85,
	FeelsLike  float64      `json:"feels_like"` // 290.71,
	Pressure   int          `json:"pressure"`   // 1021,
	Humidity   int          `json:"humidity"`   // 78,
	DewPoint   float64      `json:"dew_point"`  // 286.97,
	Uvi        float64      `json:"uvi"`        // 0.24,
	Clouds     int          `json:"clouds"`     // 20,
	Visibility int          `json:"visibility"` // 10000,
	WindSpeed  float64      `json:"wind_speed"` // 4.01,
	WindDeg    int          `json:"wind_deg"`   // 332,
	WindGust   float64      `json:"wind_gust"`  // 6.66,
	Weather    []OWMWeather `json:"weather"`    //
	Pop        float64      `json:"pop"`        // 0 (probability of precipitation)
}

// OWMFull represents the full weather data from OpenWeatherMap.
type OWMFull struct {
	Lat            float64     `json:"lat"`             // 53.18,
	Lon            float64     `json:"lon"`             // 8.6,
	Timezone       string      `json:"timezone"`        // "Europe/Berlin",
	TimezoneOffset int         `json:"timezone_offset"` // 7200,
	Hourly         []OWMHourly `json:"hourly"`
}

// Weather represents the weather data.
type Weather struct {
	Info                     string
	TempC                    float64
	Clouds                   int
	WindSpeed                float64
	Humidity                 int
	PrecipitationProbability float64
}

// Handler is a handler of retrieved weather data and querying for it.
type Handler struct {
	Data       map[model.DateAndTime]Weather
	lat, lon   string
	apiKey     string
	mutex      sync.Mutex
	queryCount int
}

// NewHandler creates a new weather handler.
func NewHandler(lat, lon, key string) *Handler {
	var h Handler
	h.lat, h.lon, h.apiKey = lat, lon, key
	return &h
}

// Update updates the weather data.
func (h *Handler) Update() error {
	// Check that we have the params we need to successfully query
	paramsProvided := (h.lat != "" && h.lon != "" && h.apiKey != "")
	if !paramsProvided {
		return fmt.Errorf("insufficient parameters for query (lat:%s,lon:%s,key-strlen:%d)",
			h.lat, h.lon, len(h.apiKey))
	}

	h.mutex.Lock()
	h.queryCount++
	owmdata, err := getHourlyInfo(h.lat, h.lon, h.apiKey)
	newData := convertHourlyDataToTimestamped(&owmdata)
	if h.Data == nil {
		h.Data = newData
	} else {
		for timestamp, data := range newData {
			h.Data[timestamp] = data
		}
	}
	h.mutex.Unlock()
	return err
}

func kelvinToCelsius(kelvin float64) (celsius float64) {
	return kelvin - 273.15
}

func convertHourlyDataToTimestamped(data *[]OWMHourly) map[model.DateAndTime]Weather {
	result := make(map[model.DateAndTime]Weather)

	for i := range *data {
		hourly := (*data)[i]
		t := time.Unix(int64(hourly.Dt), 0)

		result[*model.FromTime(t)] = Weather{
			Info:                     hourly.Weather[0].Description,
			TempC:                    kelvinToCelsius(hourly.Temp),
			Clouds:                   hourly.Clouds,
			WindSpeed:                hourly.WindSpeed,
			Humidity:                 hourly.Humidity,
			PrecipitationProbability: hourly.Pop,
		}
	}

	return result
}

func getHourlyInfo(lat, lon, apiKey string) ([]OWMHourly, error) {

	call := fmt.Sprintf("https://api.openweathermap.org/data/2.5/onecall?lat=%s&lon=%s&exclude=daily,minutely,current,alerts&appid=%s", lat, lon, apiKey)

	response, err := http.Get(call)
	if err != nil {
		return make([]OWMHourly, 0), err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return make([]OWMHourly, 0), err
	}

	data := OWMFull{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return make([]OWMHourly, 0), err
	}

	return data.Hourly, nil
}
