package model

import (
	"fmt"
	"regexp"
	"strconv"
)

type Day struct {
	Year  int
	Month int
	Day   int
}

func (d Day) Prev() Day {
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

func (d Day) Next() Day {
	if d.isLastOfMonth() {
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

func (d Day) ToString() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

func (d Day) Valid() bool {
	// basic bounds
	if d.Month < 1 ||
		d.Month > 12 ||
		d.Day < 1 ||
		d.Day > 31 {
		return false
	}

	// TODO: more sophisticated checks

	return true
}

func FromString(s string) (Day, error) {
	var result Day
	var err error

	regex := regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	parsed := regex.FindAllStringSubmatch(s, -1)

	var tmp Day
	if len(parsed) < 1 || len(parsed[0]) < 3 {
		return result, fmt.Errorf("Not enough int matches in day string '%s'", s)
	}

	year, errY := strconv.ParseInt(parsed[0][1], 10, 32)
	month, errM := strconv.ParseInt(parsed[0][2], 10, 32)
	day, errD := strconv.ParseInt(parsed[0][3], 10, 32)
	tmp = Day{int(year), int(month), int(day)}

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

func (d Day) isLastOfMonth() bool {
	if d.Month == 2 {
		if d.Day == 29 {
			return true
		}
		if d.Day == 28 && d.isLeapYear() {
			return true
		}
	}

	if d.Day == lastDaysOfMonth()[d.Month] {
		return true
	}

	return false
}

func (d Day) isLeapYear() bool {
	return d.Year%4 == 0 && (!(d.Year%100 == 0) || d.Year%400 == 0)
}
