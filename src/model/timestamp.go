package model

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
)

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

func (a Timestamp) IsAfter(b Timestamp) bool {
	if a.Hour > b.Hour {
		return true
	} else if a.Hour == b.Hour {
		return a.Minute > b.Minute
	} else {
		return false
	}
}

func (t Timestamp) Snap(res int) Timestamp {
	closestMinute := 0
	for i := 0; i <= 60; i += (60 / res) {
		distance := math.Abs(float64(i - t.Minute))
		if distance < math.Abs(float64(closestMinute-t.Minute)) {
			closestMinute = i
		}
	}
	if closestMinute == 60 {
		t.Hour += 1
		t.Minute = 0
	} else {
		t.Minute = closestMinute
	}
	return t
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

func (t1 Timestamp) DurationInMinutesUntil(t2 Timestamp) int {
	return t2.ToMinutes() - t1.ToMinutes()
}

func (t Timestamp) ToMinutes() int {
	return t.Hour*60 + t.Minute
}
