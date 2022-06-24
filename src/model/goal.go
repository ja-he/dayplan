package model

import (
	"time"
)

type Goal interface {
	Requires(Date) time.Duration
}
