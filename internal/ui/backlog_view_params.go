package ui

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// BacklogViewParams represents the zoom and scroll of a timeline  in the UI.
type BacklogViewParams struct {
	// NRowsPerHour is the number of rows in the UI that represent an hour in the
	// timeline.
	NRowsPerHour *int
	// ScrollOffset is the offset in rows by which the UI is scrolled.
	// (An unscrolled UI would have 00:00 at the very top.)
	ScrollOffset int
}

// MinutesPerRow returns the number of minutes a single row represents.
func (p *BacklogViewParams) DurationOfHeight(rows int) time.Duration {
	return time.Duration(int64(60/float64(*p.NRowsPerHour))) * time.Minute
}

func (p *BacklogViewParams) HeightOfDuration(dur time.Duration) float64 {
	return float64(*p.NRowsPerHour) * (float64(dur) / float64(time.Hour))
}

func (p *BacklogViewParams) GetScrollOffset() int { return p.ScrollOffset }
func (p *BacklogViewParams) GetZoomPercentage() float64 {
	switch *p.NRowsPerHour {
	case 6:
		return 100
	case 3:
		return 50
	case 12:
		return 200
	default:
		log.Fatal().Int("NRowsPerHour", *p.NRowsPerHour).Msg("unexpected NRowsPerHour")
		return 0
	}
}

func (p *BacklogViewParams) SetZoom(percentage float64) error {
	switch percentage {
	case 50:
		*p.NRowsPerHour = 3
	case 100:
		*p.NRowsPerHour = 6
	case 200:
		*p.NRowsPerHour = 12
	default:
		return fmt.Errorf("invalid absolute zoom percentage %f for this view-param", percentage)
	}
	return nil
}
func (p *BacklogViewParams) ChangeZoomBy(percentage float64) error {
	switch {
	case percentage == 50 && (*p.NRowsPerHour == 12 || *p.NRowsPerHour == 6):
		*p.NRowsPerHour /= 2
		return nil
	case percentage == 200 && (*p.NRowsPerHour == 6 || *p.NRowsPerHour == 3):
		*p.NRowsPerHour *= 2
		return nil
	case percentage == 100:
		return nil
	default:
		return fmt.Errorf("invalid zoom change percentage %f for this view-param", percentage)
	}
}
