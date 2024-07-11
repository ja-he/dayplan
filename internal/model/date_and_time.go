package model

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Date	represents a date, i.e. a year, month and day.
type Date struct {
	Year  int
	Month int
	Day   int
}

// DateAndTime represents a date and a time, a datetime.
type DateAndTime struct {
	Date      Date
	Timestamp Timestamp
}

// FromTime creates a DateAndTime from a time.Time.
func FromTime(t time.Time) *DateAndTime {
	return &DateAndTime{
		Date:      Date{Year: t.Year(), Month: int(t.Month()), Day: t.Day()},
		Timestamp: Timestamp{Hour: t.Hour(), Minute: t.Minute()},
	}
}

// Prev returns the previous date.
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

// Next returns the next date.
func (d Date) Next() Date {
	if d == d.GetLastOfMonth() {
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

// Backward returns a date that is `by`-many days before the receiver.
func (d Date) Backward(by int) Date {
	for i := 0; i < by; i++ {
		d = d.Prev()
	}
	return d
}

// Forward returns a date that is `by`-many days after the receiver.
func (d Date) Forward(by int) Date {
	for i := 0; i < by; i++ {
		d = d.Next()
	}
	return d
}

// ToString returns the date as a string in the format "YYYY-MM-DD".
func (d Date) ToString() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// Valid returns whether the date is valid.
// A date such as the 31st of February is invalid, for example.
func (d Date) Valid() bool {
	// verify month
	if d.Month < 1 ||
		d.Month > 12 {
		return false
	}

	if d.Day < 1 ||
		d.Day > d.GetLastOfMonth().Day {
		return false
	}

	return true
}

// FromString creates a date from a string in the format "YYYY-MM-DD".
func FromString(s string) (Date, error) {
	var result Date
	var err error

	regex := regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	parsed := regex.FindAllStringSubmatch(s, -1)

	var tmp Date
	if len(parsed) < 1 || len(parsed[0]) < 3 {
		return result, fmt.Errorf("not enough int matches in day string '%s'", s)
	}

	year, errY := strconv.ParseInt(parsed[0][1], 10, 32)
	month, errM := strconv.ParseInt(parsed[0][2], 10, 32)
	day, errD := strconv.ParseInt(parsed[0][3], 10, 32)
	tmp = Date{int(year), int(month), int(day)}

	switch {
	case errY != nil:
	case errM != nil:
	case errD != nil:
		err = fmt.Errorf("could not convert string '%s' (assuming YYYY-MM-DD format) to integers", s)
	case !tmp.Valid():
		err = fmt.Errorf("day %s (from string '%s') not valid", tmp.ToString(), s)
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

// IsAfter returns whether a date A is after a date B.
func (d Date) IsAfter(other Date) bool {
	switch {
	case d.Year < other.Year:
		return false
	case d.Year == other.Year:
		{
			switch {
			case d.Month < other.Month:
				return false
			case d.Month == other.Month:
				{
					switch {
					case d.Day < other.Day:
						return false
					case d.Day == other.Day:
						return false
					case d.Day > other.Day:
						return true
					}
				}
			case d.Month > other.Month:
				return true
			}
		}
	case d.Year > other.Year:
		return true
	}
	return false
}

// IsBefore returns whether a date A is before a date B.
func (d Date) IsBefore(other Date) bool {
	return other.IsAfter(d) && d != other
}

// DaysUntil returns the number of days from a date A until a date B is
// reached.
// (e.g. from 2021-12-14 until 2021-12-19 -> 5 days)
// expects `other` not to be before `d`
func (d Date) DaysUntil(other Date) int {
	if d.IsAfter(other) {
		panic("DaysUntil arg error: a after b")
	}

	counter := 0
	for i := d; i != other; i = i.Next() {
		counter++
	}

	return counter
}

// GetLastOfMonth returns the last date of the month of the receiver.
func (d Date) GetLastOfMonth() Date {
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

// Is returns whether the receiver is the same date as the given time.
func (d Date) Is(t time.Time) bool {
	tYear, tMonth, tDay := t.Date()
	return tYear == d.Year && int(tMonth) == d.Month && tDay == d.Day
}

// WeekBounds returns the monday and sunday of the week the receiver is in.
func (d Date) WeekBounds() (monday Date, sunday Date) {
	for d.ToWeekday() != time.Monday {
		d = d.Prev()
	}
	return d, d.Forward(6)
}

// GetDayInWeek returns the date that is on the weekday for the given index in
// the week the receiver is in.
//
// Index here means that 0 is Monday, 6 is Sunday.
func (d Date) GetDayInWeek(index int) Date {
	start, _ := d.WeekBounds()
	return start.Forward(index)
}

// GetDayInMonth returns the indexed day in the month of the receiver.
//
// Note that indexing 0 will return the first of the month.
func (d Date) GetDayInMonth(index int) Date {
	start, _ := d.MonthBounds()
	return start.Forward(index)
}

// MonthBounds returns the first and last date of the month the receiver is in.
func (d Date) MonthBounds() (first Date, last Date) {
	first = d.getFirstOfMonth()
	last = d.GetLastOfMonth()

	return first, last
}

// ToString returns the weekday as a string.
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

// ToWeekday returns the weekday of the receiver.
func (d Date) ToWeekday() time.Weekday {
	t := time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
	return t.Weekday()
}

// ToGotime returns the date as a time.Time with the time set to midnight (at the start of the day).
func (d Date) ToGotime() time.Time {
	result := time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.Now().Location())
	return result
}

type Timestamp struct {
	Hour, Minute int
}

func NewTimestampFromGotime(time time.Time) *Timestamp {
	t := Timestamp{}
	t.Hour = time.Hour()
	t.Minute = time.Minute()
	return &t
}

func NewTimestamp(s string) *Timestamp {
	components := strings.Split(s, ":")
	if len(components) != 2 {
		log.Fatalf("given string '%s' which does not fit the HH:MM format", s)
	}
	hStr := components[0]
	mStr := components[1]
	if len(hStr) != 2 || len(mStr) != 2 {
		log.Fatalf("given string '%s' which does not fit the HH:MM format", s)
	}
	h, err := strconv.Atoi(hStr)
	if err != nil {
		log.Fatalf("error converting hour string '%s' to a number", hStr)
	}
	m, err := strconv.Atoi(mStr)
	if err != nil {
		log.Fatalf("error converting minute string '%s' to a number", mStr)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		log.Fatalf("error with string-to-timestamp conversion: one of the yielded values illegal (%d) (%d)", h, m)
	}
	return &Timestamp{h, m}
}

func (a Timestamp) ToString() string {
	hPrefix := ""
	mPrefix := ""
	if a.Hour < 10 {
		hPrefix = "0"
	}
	if a.Minute < 10 {
		mPrefix = "0"
	}
	return fmt.Sprintf("%s%d:%s%d", hPrefix, a.Hour, mPrefix, a.Minute)
}

type TimeOffset struct {
	T   Timestamp
	Add bool
}

func (a Timestamp) IsBefore(b Timestamp) bool {
	if b.Hour > a.Hour {
		return true
	} else if b.Hour == a.Hour {
		return b.Minute > a.Minute
	} else {
		return false
	}
}

func (a Timestamp) IsAfter(b Timestamp) bool {
	if a.Hour > b.Hour {
		return true
	} else if a.Hour == b.Hour {
		return a.Minute > b.Minute
	} else {
		return false
	}
}

// TODO: migrate to time.Duration-based
func (t Timestamp) Snap(minutesModulus int) Timestamp {
	minutes := t.toMinutes()

	before := minutes - minutes%minutesModulus
	after := before + minutesModulus

	var resultMinutes int
	if after-minutes <= minutes-before {
		resultMinutes = after
	} else {
		resultMinutes = before
	}

	return Timestamp{
		Hour:   resultMinutes / 60,
		Minute: resultMinutes % 60,
	}
}

func (t Timestamp) Legal() bool {
	return (t.Hour < 24 && t.Minute < 60) && (t.Hour >= 0 && t.Minute >= 0)
}

func (t Timestamp) OffsetMinutes(minutes int) Timestamp {
	o := TimeOffset{}
	if minutes < 0 {
		o.Add = false
		minutes *= (-1)
	} else {
		o.Add = true
	}
	o.T.Hour = minutes / 60
	o.T.Minute = minutes % 60

	return t.Offset(o)
}

// Returns a timestamp offset by a given offset, which can be additive or
// subtractive.
// "Loops around", meaning offsetting 0:10 by -1 hour results in 23:10,
// offsetting 23:10 by +1 hour results in 0:10.
func (t Timestamp) Offset(o TimeOffset) Timestamp {
	if o.Add {
		t.Hour = (t.Hour + o.T.Hour + ((t.Minute + o.T.Minute) / 60)) % 24
		t.Minute = (t.Minute + o.T.Minute) % 60
	} else {
		extraHour := 0
		if t.Minute-o.T.Minute < 0 {
			extraHour = 1
		}
		t.Hour = (t.Hour - o.T.Hour - extraHour + 24) % 24
		t.Minute = (t.Minute - o.T.Minute + 60) % 60
	}
	return t
}

// Returns the duration in minutes between to a given timestamp t2.
// Does not check that t2 is in fact later!
func (t1 Timestamp) DurationInMinutesUntil(t2 Timestamp) int {
	return t2.toMinutes() - t1.toMinutes()
}

// Returns the duration (time.Duration) to a given timestamp t2.
// Does not check that t2 is in fact later!
func (t1 Timestamp) DurationUntil(t2 Timestamp) time.Duration {
	return t2.toGotime().Sub(t1.toGotime())
}

// toMinutes returns the number of minutes into the day (from 00:00) that this
// timestamp is.
func (t Timestamp) toMinutes() int {
	return t.Hour*60 + t.Minute
}

// toGotime returns the given timestamp as a time.Time, so only hours and
// minutes, without any date.
func (t Timestamp) toGotime() time.Time {
	return time.Time{}.
		Add(time.Duration(t.Hour) * time.Hour).
		Add(time.Duration(t.Minute) * time.Minute)
}

// DateAndTimestampToGotime returns a time.Time object from a given date and a
// given timestamp.
func DateAndTimestampToGotime(date Date, ts Timestamp) time.Time {
	return time.Date(date.Year, time.Month(date.Month), date.Day, ts.Hour, ts.Minute, 0, 0, time.UTC)
}
