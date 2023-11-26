package ui

import (
	"time"

	"github.com/ja-he/dayplan/internal/model"
)

type ViewParams interface {
	GetScrollOffset() int
	GetZoomPercentage() float64

	SetZoom(percentage float64) error
	ChangeZoomBy(percentage float64) error
}

type TimeViewParams interface {
	ViewParams
	DurationOfHeight(rows int) time.Duration
	HeightOfDuration(time.Duration) float64
}

type TimespanViewParams interface {
	TimeViewParams
	TimeAtY(int) model.Timestamp
	YForTime(model.Timestamp) int
}
