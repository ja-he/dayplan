// TODO: find a home for this
package model

import (
	"time"

	"github.com/nathan-osman/go-sunrise"
)

type SuntimesProvider struct {
	Latitude  float64
	Longitude float64
}

// SunTimes represents the sunrise and sunset times of a date.
type SunTimes struct {
	Rise, Set Timestamp
}

// GetSunTimes returns the sunrise and sunset times for the receiver-date at
// the given location.
// Warning: slow (TODO)
func (p *SuntimesProvider) Get(d Date) SunTimes {

	// calculate sunrise sunset (UTC)
	sunriseTime, sunsetTime := sunrise.SunriseSunset(p.Latitude, p.Longitude, d.Year, time.Month(d.Month), d.Day)

	// convert time to current location
	sunriseTime = sunriseTime.In(time.Now().Location())
	sunsetTime = sunsetTime.In(time.Now().Location())

	// convert to suntimes
	suntimes := SunTimes{
		*NewTimestampFromGotime(sunriseTime),
		*NewTimestampFromGotime(sunsetTime),
	}

	return suntimes
}
