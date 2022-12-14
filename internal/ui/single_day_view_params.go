package ui

import (
	"fmt"
	"time"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/rs/zerolog/log"
)

// SingleDayViewParams represents the zoom and scroll of a timeline  in the UI.
type SingleDayViewParams struct {
	// NRowsPerHour is the number of rows in the UI that represent an hour in the
	// timeline.
	NRowsPerHour int
	// ScrollOffset is the offset in rows by which the UI is scrolled.
	// (An unscrolled UI would have 00:00 at the very top.)
	ScrollOffset int
}

// MinutesPerRow returns the number of minutes a single row represents.
func (p *SingleDayViewParams) DurationOfHeight(rows int) time.Duration {
	return time.Duration(int64(60/float64(p.NRowsPerHour))) * time.Minute
}

func (p *SingleDayViewParams) HeightOfDuration(dur time.Duration) float64 {
	return float64(p.NRowsPerHour) * (float64(dur) / float64(time.Hour))
}

// TimeAtY is the time that corresponds to a given y-position.
func (p *SingleDayViewParams) TimeAtY(y int) model.Timestamp {
	minutes := y*(60/p.NRowsPerHour) + p.ScrollOffset*(60/p.NRowsPerHour)
	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}
	return ts
}

// YForTime gives the y value the given timestamp would be at with the
// receiving ViewParams.
func (p *SingleDayViewParams) YForTime(time model.Timestamp) int {
	return ((time.Hour*p.NRowsPerHour - p.ScrollOffset) + (time.Minute / (60 / p.NRowsPerHour)))
}

func (p *SingleDayViewParams) GetScrollOffset() int { return p.ScrollOffset }
func (p *SingleDayViewParams) GetZoomPercentage() float64 {
	switch p.NRowsPerHour {
	case 6:
		return 100
	case 3:
		return 50
	case 12:
		return 200
	default:
		log.Fatal().Int("NRowsPerHour", p.NRowsPerHour).Msg("unexpected NRowsPerHour")
		return 0
	}
}

func (p *SingleDayViewParams) SetZoom(percentage float64) error {
	switch percentage {
	case 50:
		p.NRowsPerHour = 3
	case 100:
		p.NRowsPerHour = 6
	case 200:
		p.NRowsPerHour = 12
	default:
		return fmt.Errorf("invalid absolute zoom percentage %f for this view-param", percentage)
	}
	return nil
}
func (p *SingleDayViewParams) ChangeZoomBy(percentage float64) error {
	switch {
	case percentage == 50 && (p.NRowsPerHour == 12 || p.NRowsPerHour == 6):
		p.NRowsPerHour /= 2
		return nil
	case percentage == 200 && (p.NRowsPerHour == 6 || p.NRowsPerHour == 3):
		p.NRowsPerHour *= 2
		return nil
	case percentage == 100:
		return nil
	default:
		return fmt.Errorf("invalid zoom change percentage %f for this view-param", percentage)
	}
}
