package storage

import "github.com/ja-he/dayplan/internal/model"

// SunTimesProvider
type SunTimesProvider interface {
	Get(model.Date) model.SunTimes
}
