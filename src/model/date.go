package model

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/kelvins/sunrisesunset"
)

type Date struct {
	Year  int
	Month int
	Day   int
}

type DayAndTime struct {
	Date      Date
	Timestamp Timestamp
}

func FromTime(t time.Time) *DayAndTime {
	return &DayAndTime{
		Date:      Date{Year: t.Year(), Month: int(t.Month()), Day: t.Day()},
		Timestamp: Timestamp{Hour: t.Hour(), Minute: t.Minute()},
	}
}

func (d Date) Prev() Date {
	if d.Day == 1 {
		if d.Month == 1 {
			d.Year--
			d.Month = 12
			d.Day = 31
		} else {
			d.Month--
			if d.Month == 2 && d.isLeapYear() {
				d.Day = 29
			} else {
				d.Day = lastDaysOfMonth()[d.Month]
			}
		}
	} else {
		d.Day--
	}
	return d
}

func (d Date) Next() Date {
	if d == d.getLastOfMonth() {
		d.Day = 1
		if d.Month == 12 {
			d.Month = 1
			d.Year++
		} else {
			d.Month++
		}
	} else {
		d.Day++
	}
	return d
}

func (d Date) Backward(by int) Date {
	for i := 0; i < by; i++ {
		d = d.Prev()
	}
	return d
}

func (d Date) Forward(by int) Date {
	for i := 0; i < by; i++ {
		d = d.Next()
	}
	return d
}

func (d Date) ToString() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

func (d Date) Valid() bool {
	// verify month
	if d.Month < 1 ||
		d.Month > 12 {
		return false
	}

	if d.Day < 1 ||
		d.Day > d.getLastOfMonth().Day {
		return false
	}

	return true
}

func FromString(s string) (Date, error) {
	var result Date
	var err error

	regex := regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	parsed := regex.FindAllStringSubmatch(s, -1)

	var tmp Date
	if len(parsed) < 1 || len(parsed[0]) < 3 {
		return result, fmt.Errorf("Not enough int matches in day string '%s'", s)
	}

	year, errY := strconv.ParseInt(parsed[0][1], 10, 32)
	month, errM := strconv.ParseInt(parsed[0][2], 10, 32)
	day, errD := strconv.ParseInt(parsed[0][3], 10, 32)
	tmp = Date{int(year), int(month), int(day)}

	switch {
	case errY != nil:
	case errM != nil:
	case errD != nil:
		err = fmt.Errorf("Could not convert string '%s' (assuming YYYY-MM-DD format) to integers", s)
	case !tmp.Valid():
		err = fmt.Errorf("Day %s (from string '%s') not valid!", tmp.ToString(), s)
	default:
		result.Day = int(day)
		result.Month = int(month)
		result.Year = int(year)
	}
	return result, err
}

func lastDaysOfMonth() map[int]int {
	return map[int]int{
		1:  31,
		2:  28,
		3:  31,
		4:  30,
		5:  31,
		6:  30,
		7:  31,
		8:  31,
		9:  30,
		10: 31,
		11: 30,
		12: 31,
	}
}

func (d Date) getFirstOfMonth() Date {
	return Date{
		Year:  d.Year,
		Month: d.Month,
		Day:   1,
	}
}

func (d Date) getLastOfMonth() Date {
	var lastDay int

	switch {
	case d.Month == 2 && d.isLeapYear():
		lastDay = 29
	default:
		lastDay = lastDaysOfMonth()[d.Month]
	}

	return Date{Year: d.Year, Month: d.Month, Day: lastDay}
}

func (d Date) isLeapYear() bool {
	return d.Year%4 == 0 && (!(d.Year%100 == 0) || d.Year%400 == 0)
}

func (d Date) Is(t time.Time) bool {
	tYear, tMonth, tDay := t.Date()
	return tYear == d.Year && int(tMonth) == d.Month && tDay == d.Day
}

// TODO: rename WeekBounds or similar
func (d Date) Week() (monday Date, sunday Date) {
	for d.ToWeekday() != time.Monday {
		d = d.Prev()
	}
	return d, d.Forward(6)
}

func (d Date) MonthBounds() (first Date, last Date) {
	first = d.getFirstOfMonth()
	last = d.getLastOfMonth()

	return first, last
}

func ToString(w time.Weekday) string {
	switch w {
	case time.Sunday:
		return "Sunday"
	case time.Monday:
		return "Monday"
	case time.Tuesday:
		return "Tuesday"
	case time.Wednesday:
		return "Wednesday"
	case time.Thursday:
		return "Thursday"
	case time.Friday:
		return "Friday"
	case time.Saturday:
		return "Saturday"
	default:
		return fmt.Sprintf("unknown: %d", int(w))
	}
}

func (d Date) ToWeekday() time.Weekday {
	t := time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
	return t.Weekday()
}

func (d Date) ToGotime() time.Time {
	result := time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.Now().Location())
	return result
}

type SunTimes struct {
	Rise, Set Timestamp
}

// Warning: slow (TODO)
func (d Date) GetSunTimes(latitude, longitude float64) (maybeSuntimes *SunTimes, err error) {
	_, utcDeltaSeconds := d.ToGotime().Zone() // TODO: does this take summer time into account if we gave it day?
	utcDeltaHours := utcDeltaSeconds / (60 * 60)

	sunriseTime, sunsetTime, err := sunrisesunset.GetSunriseSunset(latitude, longitude, float64(utcDeltaHours), d.ToGotime())

	if err == nil {
		maybeSuntimes = &SunTimes{
			*NewTimestampFromGotime(sunriseTime),
			*NewTimestampFromGotime(sunsetTime),
		}
	} else {
		maybeSuntimes = nil
	}

	return maybeSuntimes, err
}
