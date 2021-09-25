package weather

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"dayplan/src/model"
)

type OwmWeather struct {
	Id          int    `json:"id"`          // 801,
	Main        string `json:"main"`        // "Clouds",
	Description string `json:"description"` // "few clouds",
	Icon        string `json:"icon"`        // "02d"
}

type OwmHourly struct {
	Dt         uint64       `json:"dt"`         // 1630429200,
	Temp       float64      `json:"temp"`       // 290.85,
	Feels_like float64      `json:"feels_like"` // 290.71,
	Pressure   int          `json:"pressure"`   // 1021,
	Humidity   int          `json:"humidity"`   // 78,
	Dew_point  float64      `json:"dew_point"`  // 286.97,
	Uvi        float64      `json:"uvi"`        // 0.24,
	Clouds     int          `json:"clouds"`     // 20,
	Visibility int          `json:"visibility"` // 10000,
	Wind_speed float64      `json:"wind_speed"` // 4.01,
	Wind_deg   int          `json:"wind_deg"`   // 332,
	Wind_gust  float64      `json:"wind_gust"`  // 6.66,
	Weather    []OwmWeather `json:"weather"`    //
	Pop        float64      `json:"pop"`        // 0 (probability of precipitation)
}

type OwmFull struct {
	Lat             float64     `json:"lat"`             // 53.18,
	Lon             float64     `json:"lon"`             // 8.6,
	Timezone        string      `json:"timezone"`        // "Europe/Berlin",
	Timezone_offset int         `json:"timezone_offset"` // 7200,
	Hourly          []OwmHourly `json:"hourly"`
}

type MyWeather struct {
	Info                     string
	TempC                    float64
	Clouds                   int
	WindSpeed                float64
	Humidity                 int
	PrecipitationProbability float64
}

type Handler struct {
	Data     map[model.Timestamp]MyWeather
	lat, lon string
	apiKey   string
}

func NewHandler(lat, lon, key string) *Handler {
	var h Handler
	h.lat, h.lon, h.apiKey = lat, lon, key
	return &h
}

func (h *Handler) Update() {
	owmdata := GetHourlyInfo(h.lat, h.lon, h.apiKey)
	newData := GetTodaysWeather(&owmdata)
	if h.Data == nil {
		h.Data = newData
	} else {
		for timestamp, data := range newData {
			h.Data[timestamp] = data
		}
	}
}

func isToday(t time.Time) bool {
	now := time.Now()
	thisYear, thisMonth, thisDay := now.Date()

	qYear, qMonth, qDay := t.Date()

	return thisYear == qYear && thisMonth == qMonth && thisDay == qDay
}

func kelvinToCelsius(kelvin float64) (celsius float64) {
	return kelvin - 273.15
}

func GetTodaysWeather(data *[]OwmHourly) map[model.Timestamp]MyWeather {
	result := make(map[model.Timestamp]MyWeather)

	for i := range *data {
		hourly := (*data)[i]
		t := time.Unix(int64(hourly.Dt), 0)

		if isToday(t) {
			result[model.Timestamp{Hour: t.Hour(), Minute: t.Minute()}] = MyWeather{
				Info:                     hourly.Weather[0].Description,
				TempC:                    kelvinToCelsius(hourly.Temp),
				Clouds:                   hourly.Clouds,
				WindSpeed:                hourly.Wind_speed,
				Humidity:                 hourly.Humidity,
				PrecipitationProbability: hourly.Pop,
			}
		}
	}

	return result
}

func GetHourlyInfo(lat, lon, apiKey string) []OwmHourly {

	call := fmt.Sprintf("https://api.openweathermap.org/data/2.5/onecall?lat=%s&lon=%s&exclude=daily,minutely,current,alerts&appid=%s", lat, lon, apiKey)

	response, err := http.Get(call)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		panic(err)
	}

	data := OwmFull{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		panic(err)
	}

	return data.Hourly
}
