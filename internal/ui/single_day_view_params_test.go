package ui_test

import (
	"testing"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestSingleDayViewParamsYForTime(t *testing.T) {
	vp := &ui.SingleDayViewParams{
		NRowsPerHour: 6,
		ScrollOffset: 0,
	}

	testData := map[model.Timestamp]int{
		{Hour: 0, Minute: 0}:  0,
		{Hour: 0, Minute: 1}:  0,
		{Hour: 0, Minute: 9}:  0,
		{Hour: 0, Minute: 10}: 1,
		{Hour: 6, Minute: 0}:  36,
		{Hour: 6, Minute: 18}: 37,
		{Hour: 6, Minute: 20}: 38,
	}

	for timeInput, expectedResult := range testData {
		result := vp.YForTime(timeInput)
		assert.Equal(t, result, expectedResult, "Computed Y (%d) for timestamp %s wrong (expected %d).", result, timeInput.ToString(), expectedResult)
	}

}
