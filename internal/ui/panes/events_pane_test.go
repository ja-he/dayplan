package panes

import (
	"testing"
	"time"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/rs/zerolog"
)

// testViewParams is a simple implementation of TimespanViewParams for testing.
// It uses 1 row per 10 minutes (6 rows per hour) with no scroll offset.
type testViewParams struct {
	rowsPerHour  int
	scrollOffset int
}

func (p *testViewParams) GetScrollOffset() int               { return p.scrollOffset }
func (p *testViewParams) GetZoomPercentage() float64         { return 100 }
func (p *testViewParams) SetZoom(float64) error              { return nil }
func (p *testViewParams) ChangeZoomBy(float64) error         { return nil }
func (p *testViewParams) DurationOfHeight(int) time.Duration { return 10 * time.Minute }
func (p *testViewParams) HeightOfDuration(d time.Duration) float64 {
	return float64(p.rowsPerHour) * (float64(d) / float64(time.Hour))
}
func (p *testViewParams) TimeAtY(y int) model.Timestamp {
	minutesPerRow := 60 / p.rowsPerHour
	minutes := y*minutesPerRow + p.scrollOffset*minutesPerRow
	return model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}
}
func (p *testViewParams) YForTime(ts model.Timestamp) int {
	minutesPerRow := 60 / p.rowsPerHour
	return (ts.Hour*p.rowsPerHour - p.scrollOffset) + (ts.Minute / minutesPerRow)
}

// newTestEventsPane creates a minimal EventsPane for testing computeRects.
func newTestEventsPane(viewParams ui.TimespanViewParams, getCurrentEventID func() *model.EventID) *EventsPane {
	return &EventsPane{
		viewParams:        viewParams,
		getCurrentEventID: getCurrentEventID,
		log:               zerolog.Nop(),
	}
}

// helper to create an event with start and end timestamps on a given date.
// Uses time.Local since computeRects converts to Local timezone.
func makeEvent(date model.Date, startHour, startMin, endHour, endMin int, name string) *model.Event {
	loc := time.Local
	start := time.Date(date.Year, time.Month(date.Month), date.Day, startHour, startMin, 0, 0, loc)
	end := time.Date(date.Year, time.Month(date.Month), date.Day, endHour, endMin, 0, 0, loc)
	return &model.Event{
		ID:       model.EventID(name),
		Name:     name,
		Category: "test",
		Start:    start,
		End:      end,
	}
}

// helper to create a multi-day event.
// Uses time.Local since computeRects converts to Local timezone.
func makeMultiDayEvent(startDate model.Date, startHour, startMin int, endDate model.Date, endHour, endMin int, name string) *model.Event {
	loc := time.Local
	start := time.Date(startDate.Year, time.Month(startDate.Month), startDate.Day, startHour, startMin, 0, 0, loc)
	end := time.Date(endDate.Year, time.Month(endDate.Month), endDate.Day, endHour, endMin, 0, 0, loc)
	return &model.Event{
		ID:       model.EventID(name),
		Name:     name,
		Category: "test",
		Start:    start,
		End:      end,
	}
}

func TestComputeRects_EmptyEventList(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	eventList := &model.EventList{Events: []*model.Event{}}
	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 0 {
		t.Errorf("expected empty map for empty event list, got %d entries", len(result))
	}
}

func TestComputeRects_SingleEvent(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	// 6 rows per hour means 1 row = 10 minutes
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event from 10:00 to 11:00 (1 hour = 6 rows)
	event := makeEvent(date, 10, 0, 11, 0, "meeting")
	eventList := &model.EventList{Events: []*model.Event{event}}

	offsetX, offsetY, width, height := 10, 0, 100, 144
	result := pane.computeRects(date, eventList, offsetX, offsetY, width, height)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(result))
	}

	rect, ok := result[event]
	if !ok {
		t.Fatal("expected rect for the event")
	}

	// Y should be at row 60 (10 hours * 6 rows/hour)
	expectedY := 60
	if rect.Y != expectedY {
		t.Errorf("expected Y=%d, got Y=%d", expectedY, rect.Y)
	}

	// Height should be 6 rows (1 hour)
	expectedH := 6
	if rect.H != expectedH {
		t.Errorf("expected H=%d, got H=%d", expectedH, rect.H)
	}

	// Width should be full width
	if rect.W != width {
		t.Errorf("expected W=%d (full width), got W=%d", width, rect.W)
	}

	// X should be at offsetX
	if rect.X != offsetX {
		t.Errorf("expected X=%d, got X=%d", offsetX, rect.X)
	}
}

func TestComputeRects_TwoNonOverlappingEvents(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event 1: 09:00-10:00, Event 2: 11:00-12:00
	event1 := makeEvent(date, 9, 0, 10, 0, "morning")
	event2 := makeEvent(date, 11, 0, 12, 0, "midday")
	eventList := &model.EventList{Events: []*model.Event{event1, event2}}

	offsetX, width := 0, 100
	result := pane.computeRects(date, eventList, offsetX, 0, width, 144)

	if len(result) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(result))
	}

	rect1 := result[event1]
	rect2 := result[event2]

	// Both should have full width since they don't overlap
	if rect1.W != width {
		t.Errorf("event1: expected W=%d, got W=%d", width, rect1.W)
	}
	if rect2.W != width {
		t.Errorf("event2: expected W=%d, got W=%d", width, rect2.W)
	}

	// Check Y positions
	if rect1.Y != 54 { // 9 * 6
		t.Errorf("event1: expected Y=54, got Y=%d", rect1.Y)
	}
	if rect2.Y != 66 { // 11 * 6
		t.Errorf("event2: expected Y=66, got Y=%d", rect2.Y)
	}
}

func TestComputeRects_TwoOverlappingEvents(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event 1: 09:00-11:00, Event 2: 10:00-12:00 (overlap at 10:00-11:00)
	event1 := makeEvent(date, 9, 0, 11, 0, "first")
	event2 := makeEvent(date, 10, 0, 12, 0, "second")
	eventList := &model.EventList{Events: []*model.Event{event1, event2}}

	offsetX, width := 0, 100
	result := pane.computeRects(date, eventList, offsetX, 0, width, 144)

	if len(result) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(result))
	}

	rect1 := result[event1]
	rect2 := result[event2]

	// First event should have full width
	if rect1.W != width {
		t.Errorf("event1: expected W=%d (full width), got W=%d", width, rect1.W)
	}

	// Second event should be narrower (75% of width)
	if rect2.W >= rect1.W {
		t.Errorf("overlapping event2 should be narrower than event1: got W1=%d, W2=%d", rect1.W, rect2.W)
	}

	// Second event should be positioned further right
	if rect2.X <= rect1.X {
		t.Errorf("overlapping event2 should be further right than event1: got X1=%d, X2=%d", rect1.X, rect2.X)
	}

	// Heights should match their durations
	if rect1.H != 12 { // 2 hours = 12 rows
		t.Errorf("event1: expected H=12, got H=%d", rect1.H)
	}
	if rect2.H != 12 { // 2 hours = 12 rows
		t.Errorf("event2: expected H=12, got H=%d", rect2.H)
	}
}

func TestComputeRects_ThreeOverlappingEvents(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Three events that all overlap
	event1 := makeEvent(date, 9, 0, 12, 0, "first")
	event2 := makeEvent(date, 10, 0, 12, 0, "second")
	event3 := makeEvent(date, 11, 0, 12, 0, "third")
	eventList := &model.EventList{Events: []*model.Event{event1, event2, event3}}

	offsetX, width := 0, 100
	result := pane.computeRects(date, eventList, offsetX, 0, width, 144)

	if len(result) != 3 {
		t.Fatalf("expected 3 rects, got %d", len(result))
	}

	rect1 := result[event1]
	rect2 := result[event2]
	rect3 := result[event3]

	// Widths should decrease progressively
	if rect2.W >= rect1.W {
		t.Errorf("event2 should be narrower than event1: W1=%d, W2=%d", rect1.W, rect2.W)
	}
	if rect3.W >= rect2.W {
		t.Errorf("event3 should be narrower than event2: W2=%d, W3=%d", rect2.W, rect3.W)
	}

	// X positions should increase (move right)
	if rect2.X <= rect1.X {
		t.Errorf("event2 should be further right than event1: X1=%d, X2=%d", rect1.X, rect2.X)
	}
	if rect3.X <= rect2.X {
		t.Errorf("event3 should be further right than event2: X2=%d, X3=%d", rect2.X, rect3.X)
	}
}

func TestComputeRects_MultiDayEventStartsBefore(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event starts on previous day at 22:00, ends on this day at 08:00
	prevDate := date.Prev()
	event := makeMultiDayEvent(prevDate, 22, 0, date, 8, 0, "overnight")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(result))
	}

	rect := result[event]

	// Y should be 0 (starts at midnight since actual start is on previous day)
	if rect.Y != 0 {
		t.Errorf("expected Y=0 for event starting before the date, got Y=%d", rect.Y)
	}

	// Height should be 48 rows (8 hours from 00:00 to 08:00)
	expectedH := 48
	if rect.H != expectedH {
		t.Errorf("expected H=%d, got H=%d", expectedH, rect.H)
	}
}

func TestComputeRects_MultiDayEventEndsAfter(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event starts on this day at 20:00, ends on next day at 06:00
	nextDate := date.Next()
	event := makeMultiDayEvent(date, 20, 0, nextDate, 6, 0, "overnight")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(result))
	}

	rect := result[event]

	// Y should be at row 120 (20 * 6)
	expectedY := 120
	if rect.Y != expectedY {
		t.Errorf("expected Y=%d, got Y=%d", expectedY, rect.Y)
	}

	// Height should be 24 rows (4 hours from 20:00 to 24:00)
	expectedH := 24
	if rect.H != expectedH {
		t.Errorf("expected H=%d, got H=%d", expectedH, rect.H)
	}
}

func TestComputeRects_VeryShortEvent(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event of only 5 minutes (less than 1 row at 10 min/row)
	event := makeEvent(date, 10, 0, 10, 5, "quick")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(result))
	}

	rect := result[event]

	// Height should be at least 1 (minimum)
	if rect.H < 1 {
		t.Errorf("expected H >= 1 for very short event, got H=%d", rect.H)
	}
}

func TestComputeRects_EventsUnstackWhenNoLongerOverlapping(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event 1: 09:00-10:00
	// Event 2: 09:30-10:30 (overlaps with event 1)
	// Event 3: 11:00-12:00 (does NOT overlap with events 1 or 2)
	event1 := makeEvent(date, 9, 0, 10, 0, "first")
	event2 := makeEvent(date, 9, 30, 10, 30, "second")
	event3 := makeEvent(date, 11, 0, 12, 0, "third")
	eventList := &model.EventList{Events: []*model.Event{event1, event2, event3}}

	offsetX, width := 0, 100
	result := pane.computeRects(date, eventList, offsetX, 0, width, 144)

	if len(result) != 3 {
		t.Fatalf("expected 3 rects, got %d", len(result))
	}

	rect1 := result[event1]
	rect2 := result[event2]
	rect3 := result[event3]

	// Event 1 and 2 overlap, so event 2 should be narrower
	if rect2.W >= rect1.W {
		t.Errorf("event2 should be narrower than event1 (they overlap): W1=%d, W2=%d", rect1.W, rect2.W)
	}

	// Event 3 doesn't overlap with anything anymore, should get full width again
	if rect3.W != width {
		t.Errorf("event3 should have full width (no overlap): expected W=%d, got W=%d", width, rect3.W)
	}

	// Event 3's X should be back at offsetX
	if rect3.X != offsetX {
		t.Errorf("event3 should be at X=%d (unstacked), got X=%d", offsetX, rect3.X)
	}
}

func TestComputeRects_CurrentEventIsWider(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}

	event := makeEvent(date, 10, 0, 11, 0, "current")
	currentID := event.ID
	pane := newTestEventsPane(viewParams, func() *model.EventID { return &currentID })

	eventList := &model.EventList{Events: []*model.Event{event}}

	offsetX, width := 10, 100
	result := pane.computeRects(date, eventList, offsetX, 0, width, 144)

	rect := result[event]

	// Current event should be wider by 2 and shifted left by 1
	expectedW := width + 2
	expectedX := offsetX - 1
	if rect.W != expectedW {
		t.Errorf("current event should be wider: expected W=%d, got W=%d", expectedW, rect.W)
	}
	if rect.X != expectedX {
		t.Errorf("current event should be shifted left: expected X=%d, got X=%d", expectedX, rect.X)
	}
}

func TestComputeRects_ContainedEvent(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event 1: 09:00-12:00 (3 hours)
	// Event 2: 10:00-11:00 (contained within event 1)
	event1 := makeEvent(date, 9, 0, 12, 0, "outer")
	event2 := makeEvent(date, 10, 0, 11, 0, "inner")
	eventList := &model.EventList{Events: []*model.Event{event1, event2}}

	offsetX, width := 0, 100
	result := pane.computeRects(date, eventList, offsetX, 0, width, 144)

	rect1 := result[event1]
	rect2 := result[event2]

	// Outer event should have full width
	if rect1.W != width {
		t.Errorf("outer event should have full width: expected W=%d, got W=%d", width, rect1.W)
	}

	// Inner event should be narrower and offset to the right
	if rect2.W >= rect1.W {
		t.Errorf("inner event should be narrower: W_outer=%d, W_inner=%d", rect1.W, rect2.W)
	}
	if rect2.X <= rect1.X {
		t.Errorf("inner event should be further right: X_outer=%d, X_inner=%d", rect1.X, rect2.X)
	}

	// Heights should match their durations
	if rect1.H != 18 { // 3 hours = 18 rows
		t.Errorf("outer event: expected H=18, got H=%d", rect1.H)
	}
	if rect2.H != 6 { // 1 hour = 6 rows
		t.Errorf("inner event: expected H=6, got H=%d", rect2.H)
	}
}

func TestComputeRects_PositionsRespectOffsets(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	event := makeEvent(date, 10, 0, 11, 0, "test")
	eventList := &model.EventList{Events: []*model.Event{event}}

	offsetX, offsetY, width := 50, 20, 100
	result := pane.computeRects(date, eventList, offsetX, offsetY, width, 144)

	rect := result[event]

	// X should be at offsetX
	if rect.X != offsetX {
		t.Errorf("expected X=%d, got X=%d", offsetX, rect.X)
	}

	// Y should be (10 * 6) + offsetY = 60 + 20 = 80
	expectedY := 60 + offsetY
	if rect.Y != expectedY {
		t.Errorf("expected Y=%d, got Y=%d", expectedY, rect.Y)
	}
}

func TestComputeRects_WithScrollOffset(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	// Scroll offset of 30 means we've scrolled down 30 rows (5 hours at 6 rows/hour)
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 30}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event at 10:00-11:00
	event := makeEvent(date, 10, 0, 11, 0, "test")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	rect := result[event]

	// With scroll offset of 30, 10:00 (normally at row 60) should be at row 60 - 30 = 30
	expectedY := 30
	if rect.Y != expectedY {
		t.Errorf("expected Y=%d with scroll offset, got Y=%d", expectedY, rect.Y)
	}
}

func TestComputeRects_PreservesEventPointers(t *testing.T) {
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	event1 := makeEvent(date, 9, 0, 10, 0, "first")
	event2 := makeEvent(date, 11, 0, 12, 0, "second")
	eventList := &model.EventList{Events: []*model.Event{event1, event2}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	// Verify the returned map uses the exact same pointers
	if _, ok := result[event1]; !ok {
		t.Error("result should contain the exact event1 pointer as key")
	}
	if _, ok := result[event2]; !ok {
		t.Error("result should contain the exact event2 pointer as key")
	}
}

// Edge case tests for events that are only partially on the date

func TestComputeRects_PreviousDayEventEndingEarlyMorning(t *testing.T) {
	// Event from previous day ending at 00:50 on the current date
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event: yesterday 22:00 to today 00:50
	event := makeMultiDayEvent(prevDate, 22, 0, date, 0, 50, "overnight")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(result))
	}

	rect := result[event]

	// Y should be 0 (starts at midnight)
	if rect.Y != 0 {
		t.Errorf("expected Y=0, got Y=%d", rect.Y)
	}

	// Height should be 5 rows (50 minutes at 6 rows/hour = 50/10 = 5)
	expectedH := 5
	if rect.H != expectedH {
		t.Errorf("expected H=%d (50 minutes), got H=%d", expectedH, rect.H)
	}

	// Should have full width (no overlapping events)
	if rect.W != 100 {
		t.Errorf("expected W=100, got W=%d", rect.W)
	}
}

func TestComputeRects_PreviousDayEventEndingAtMidnight(t *testing.T) {
	// Event from previous day ending exactly at midnight (00:00)
	// This event should NOT be rendered on the current date since it ends at the boundary
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event: yesterday 22:00 to today 00:00 (ends exactly at midnight)
	event := makeMultiDayEvent(prevDate, 22, 0, date, 0, 0, "until_midnight")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect (event ends at 00:00 on this date), got %d", len(result))
	}

	rect := result[event]

	// Y should be 0
	if rect.Y != 0 {
		t.Errorf("expected Y=0, got Y=%d", rect.Y)
	}

	// Height should be at least 1 (minimum height for zero-duration visible portion)
	if rect.H < 1 {
		t.Errorf("expected H >= 1 (minimum), got H=%d", rect.H)
	}
}

func TestComputeRects_PreviousDayEventWithSameDayEvent(t *testing.T) {
	// Event from previous day ending at 00:50, plus an event starting at 09:00
	// These should NOT overlap, so both should have full width
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	overnight := makeMultiDayEvent(prevDate, 22, 0, date, 0, 50, "overnight")
	morning := makeEvent(date, 9, 0, 10, 0, "morning")
	eventList := &model.EventList{Events: []*model.Event{overnight, morning}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(result))
	}

	rectOvernight := result[overnight]
	rectMorning := result[morning]

	// Both should have full width since they don't overlap
	if rectOvernight.W != 100 {
		t.Errorf("overnight event: expected W=100, got W=%d", rectOvernight.W)
	}
	if rectMorning.W != 100 {
		t.Errorf("morning event: expected W=100, got W=%d", rectMorning.W)
	}

	// Overnight rect should be at Y=0
	if rectOvernight.Y != 0 {
		t.Errorf("overnight event: expected Y=0, got Y=%d", rectOvernight.Y)
	}

	// Morning rect should be at Y=54 (9 hours * 6 rows/hour)
	if rectMorning.Y != 54 {
		t.Errorf("morning event: expected Y=54, got Y=%d", rectMorning.Y)
	}
}

func TestComputeRects_PreviousDayEventOverlappingWithEarlyMorningEvent(t *testing.T) {
	// Event from previous day ending at 01:00, overlapping with event starting at 00:30
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	overnight := makeMultiDayEvent(prevDate, 22, 0, date, 1, 0, "overnight")
	earlyMorning := makeEvent(date, 0, 30, 1, 30, "early_morning")
	eventList := &model.EventList{Events: []*model.Event{overnight, earlyMorning}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(result))
	}

	rectOvernight := result[overnight]
	rectEarlyMorning := result[earlyMorning]

	// Overnight should have full width (it's first)
	if rectOvernight.W != 100 {
		t.Errorf("overnight event: expected W=100 (full width), got W=%d", rectOvernight.W)
	}

	// Early morning event should be narrower (it overlaps with overnight)
	if rectEarlyMorning.W >= rectOvernight.W {
		t.Errorf("early morning event should be narrower than overnight: W_overnight=%d, W_early=%d",
			rectOvernight.W, rectEarlyMorning.W)
	}

	// Early morning should be offset to the right
	if rectEarlyMorning.X <= rectOvernight.X {
		t.Errorf("early morning event should be further right: X_overnight=%d, X_early=%d",
			rectOvernight.X, rectEarlyMorning.X)
	}
}

func TestComputeRects_EventSpanningEntireDay(t *testing.T) {
	// Event that starts before the date and ends after the date
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	nextDate := date.Next()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event: yesterday 20:00 to tomorrow 08:00 (spans entire current day)
	event := makeMultiDayEvent(prevDate, 20, 0, nextDate, 8, 0, "multi_day")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(result))
	}

	rect := result[event]

	// Y should be 0 (clamped to 00:00)
	if rect.Y != 0 {
		t.Errorf("expected Y=0, got Y=%d", rect.Y)
	}

	// Height should be 144 (full day: 24 hours * 6 rows/hour)
	expectedH := 144
	if rect.H != expectedH {
		t.Errorf("expected H=%d (full day), got H=%d", expectedH, rect.H)
	}
}

func TestComputeRects_LateNightEventEndingNextDay(t *testing.T) {
	// Event starting late at night (23:30) and ending next day (01:00)
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	nextDate := date.Next()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	event := makeMultiDayEvent(date, 23, 30, nextDate, 1, 0, "late_night")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(result))
	}

	rect := result[event]

	// Y should be at 23:30 = 23*6 + 3 = 141
	expectedY := 141
	if rect.Y != expectedY {
		t.Errorf("expected Y=%d, got Y=%d", expectedY, rect.Y)
	}

	// Height should be 3 rows (30 minutes from 23:30 to 24:00)
	expectedH := 3
	if rect.H != expectedH {
		t.Errorf("expected H=%d (30 minutes to midnight), got H=%d", expectedH, rect.H)
	}
}

func TestComputeRects_EventStartingAtMidnight(t *testing.T) {
	// Event starting exactly at midnight
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	event := makeEvent(date, 0, 0, 1, 0, "midnight_start")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(result))
	}

	rect := result[event]

	// Y should be 0
	if rect.Y != 0 {
		t.Errorf("expected Y=0, got Y=%d", rect.Y)
	}

	// Height should be 6 (1 hour)
	if rect.H != 6 {
		t.Errorf("expected H=6, got H=%d", rect.H)
	}
}

func TestComputeRects_TwoEventsFromPreviousDayOverlapping(t *testing.T) {
	// Two events from previous day, both ending early morning, overlapping
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event 1: yesterday 20:00 to today 02:00
	// Event 2: yesterday 22:00 to today 01:00
	event1 := makeMultiDayEvent(prevDate, 20, 0, date, 2, 0, "first_overnight")
	event2 := makeMultiDayEvent(prevDate, 22, 0, date, 1, 0, "second_overnight")
	eventList := &model.EventList{Events: []*model.Event{event1, event2}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(result))
	}

	rect1 := result[event1]
	rect2 := result[event2]

	// Both should start at Y=0 (both clamped to midnight)
	if rect1.Y != 0 {
		t.Errorf("event1: expected Y=0, got Y=%d", rect1.Y)
	}
	if rect2.Y != 0 {
		t.Errorf("event2: expected Y=0, got Y=%d", rect2.Y)
	}

	// Event1 (starting earlier) should have full width
	if rect1.W != 100 {
		t.Errorf("event1: expected W=100, got W=%d", rect1.W)
	}

	// Event2 should be narrower (overlaps with event1)
	if rect2.W >= rect1.W {
		t.Errorf("event2 should be narrower than event1: W1=%d, W2=%d", rect1.W, rect2.W)
	}

	// Heights should reflect their respective end times
	// Event1 ends at 02:00 = 12 rows
	if rect1.H != 12 {
		t.Errorf("event1: expected H=12 (2 hours), got H=%d", rect1.H)
	}
	// Event2 ends at 01:00 = 6 rows
	if rect2.H != 6 {
		t.Errorf("event2: expected H=6 (1 hour), got H=%d", rect2.H)
	}
}

func TestComputeRects_PreviousDayEventUnstacksForLaterEvent(t *testing.T) {
	// Previous day event ending at 00:50, then a gap, then an event at 09:00
	// The 09:00 event should NOT be stacked under the overnight event
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	overnight := makeMultiDayEvent(prevDate, 22, 0, date, 0, 50, "overnight")
	morning := makeEvent(date, 9, 0, 10, 0, "morning")
	eventList := &model.EventList{Events: []*model.Event{overnight, morning}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	rectMorning := result[morning]

	// Morning event should have full width (overnight has ended)
	if rectMorning.W != 100 {
		t.Errorf("morning event should have full width after overnight ends: got W=%d", rectMorning.W)
	}

	// Morning event should be at X=0 (not offset)
	if rectMorning.X != 0 {
		t.Errorf("morning event should be at X=0 (unstacked): got X=%d", rectMorning.X)
	}
}

func TestComputeRects_VeryShortOvernightEvent(t *testing.T) {
	// Event from previous day ending just 5 minutes into current day
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event: yesterday 23:55 to today 00:05 (10 minutes total, 5 on each day)
	event := makeMultiDayEvent(prevDate, 23, 55, date, 0, 5, "brief_overnight")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(result))
	}

	rect := result[event]

	// Y should be 0
	if rect.Y != 0 {
		t.Errorf("expected Y=0, got Y=%d", rect.Y)
	}

	// Height should be at least 1 (5 minutes < 10 min/row, but minimum height is 1)
	if rect.H < 1 {
		t.Errorf("expected H >= 1 (minimum), got H=%d", rect.H)
	}
}

func TestComputeRects_EventEndingExactlyAtEndOfDay(t *testing.T) {
	// Event ending exactly at 24:00 (midnight of next day)
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	nextDate := date.Next()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event: today 22:00 to tomorrow 00:00
	event := makeMultiDayEvent(date, 22, 0, nextDate, 0, 0, "until_midnight")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(result))
	}

	rect := result[event]

	// Y should be at 22:00 = 132
	expectedY := 132
	if rect.Y != expectedY {
		t.Errorf("expected Y=%d, got Y=%d", expectedY, rect.Y)
	}

	// Height should be 12 (2 hours from 22:00 to 24:00)
	expectedH := 12
	if rect.H != expectedH {
		t.Errorf("expected H=%d, got H=%d", expectedH, rect.H)
	}
}

func TestComputeRects_OvernightEventWithScrollOffset(t *testing.T) {
	// Previous day event with a scroll offset that could cause negative Y
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	// Scroll offset of 6 rows = 1 hour scrolled down
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 6}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// Event: yesterday 22:00 to today 00:30
	event := makeMultiDayEvent(prevDate, 22, 0, date, 0, 30, "overnight")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	rect := result[event]

	// With scroll offset 6, Y for 00:00 would be 0 - 6 = -6
	// The visible portion starts at 00:00 clamped, so Y = -6
	expectedY := -6
	if rect.Y != expectedY {
		t.Errorf("expected Y=%d (negative due to scroll), got Y=%d", expectedY, rect.Y)
	}

	// Height should still be 3 rows (30 minutes)
	expectedH := 3
	if rect.H != expectedH {
		t.Errorf("expected H=%d, got H=%d", expectedH, rect.H)
	}
}

func TestComputeRects_PreviousDayEventEndingAt0010(t *testing.T) {
	// Specific test for event ending at 00:10 (exactly 1 row at 6 rows/hour)
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	event := makeMultiDayEvent(prevDate, 23, 0, date, 0, 10, "overnight")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	rect := result[event]

	if rect.Y != 0 {
		t.Errorf("expected Y=0, got Y=%d", rect.Y)
	}

	// 10 minutes = 1 row
	if rect.H != 1 {
		t.Errorf("expected H=1 (10 minutes), got H=%d", rect.H)
	}
}

func TestComputeRects_PreviousDayEventEndingAt0015(t *testing.T) {
	// Test for event ending at 00:15 (1.5 rows, should round to 1)
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	event := makeMultiDayEvent(prevDate, 23, 0, date, 0, 15, "overnight")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	rect := result[event]

	if rect.Y != 0 {
		t.Errorf("expected Y=0, got Y=%d", rect.Y)
	}

	// 15 minutes at 10 min/row = 1.5, YForTime rounds down so H = 1
	if rect.H != 1 {
		t.Errorf("expected H=1 (15 minutes truncates to 1 row), got H=%d", rect.H)
	}
}

func TestComputeRects_EventsAroundMidnightBoundary(t *testing.T) {
	// Event ending at 00:01 (just past midnight)
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	event := makeMultiDayEvent(prevDate, 23, 50, date, 0, 1, "brief_overnight")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	rect := result[event]

	// Should be at Y=0, clamped start
	if rect.Y != 0 {
		t.Errorf("expected Y=0, got Y=%d", rect.Y)
	}

	// 1 minute is less than 10 min/row, but minimum height is 1
	if rect.H < 1 {
		t.Errorf("expected H >= 1 (minimum), got H=%d", rect.H)
	}
}

func TestComputeRects_HeightCalculation(t *testing.T) {
	// Verify height calculation is correct: endY - startY
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	// 90 minute event from 10:00 to 11:30
	event := makeEvent(date, 10, 0, 11, 30, "ninety_minutes")
	eventList := &model.EventList{Events: []*model.Event{event}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	rect := result[event]

	// Y = 10 * 6 = 60
	if rect.Y != 60 {
		t.Errorf("expected Y=60, got Y=%d", rect.Y)
	}

	// H = YForTime(11:30) - YForTime(10:00) = (11*6 + 3) - 60 = 69 - 60 = 9
	expectedH := 9
	if rect.H != expectedH {
		t.Errorf("expected H=%d (90 minutes), got H=%d", expectedH, rect.H)
	}
}

func TestComputeRects_MixedEventsComplexScenario(t *testing.T) {
	// Complex scenario: overnight event, early morning event, gap, then later events
	date := model.Date{Year: 2025, Month: 1, Day: 15}
	prevDate := date.Prev()
	viewParams := &testViewParams{rowsPerHour: 6, scrollOffset: 0}
	pane := newTestEventsPane(viewParams, func() *model.EventID { return nil })

	overnight := makeMultiDayEvent(prevDate, 22, 0, date, 0, 45, "sleep")
	breakfast := makeEvent(date, 7, 0, 7, 30, "breakfast")
	work := makeEvent(date, 9, 0, 17, 0, "work")

	// Note: events should be in order by original start time
	eventList := &model.EventList{Events: []*model.Event{overnight, breakfast, work}}

	result := pane.computeRects(date, eventList, 0, 0, 100, 144)

	if len(result) != 3 {
		t.Fatalf("expected 3 rects, got %d", len(result))
	}

	rectOvernight := result[overnight]
	rectBreakfast := result[breakfast]
	rectWork := result[work]

	// All should have full width (no overlaps)
	if rectOvernight.W != 100 {
		t.Errorf("overnight: expected W=100, got W=%d", rectOvernight.W)
	}
	if rectBreakfast.W != 100 {
		t.Errorf("breakfast: expected W=100, got W=%d", rectBreakfast.W)
	}
	if rectWork.W != 100 {
		t.Errorf("work: expected W=100, got W=%d", rectWork.W)
	}

	// Verify Y positions
	if rectOvernight.Y != 0 {
		t.Errorf("overnight: expected Y=0, got Y=%d", rectOvernight.Y)
	}
	if rectBreakfast.Y != 42 { // 7 * 6
		t.Errorf("breakfast: expected Y=42, got Y=%d", rectBreakfast.Y)
	}
	if rectWork.Y != 54 { // 9 * 6
		t.Errorf("work: expected Y=54, got Y=%d", rectWork.Y)
	}

	// Verify heights
	// overnight: 45 minutes = 4 rows (45/10 truncated)
	if rectOvernight.H != 4 {
		t.Errorf("overnight: expected H=4, got H=%d", rectOvernight.H)
	}
	// breakfast: 30 minutes = 3 rows
	if rectBreakfast.H != 3 {
		t.Errorf("breakfast: expected H=3, got H=%d", rectBreakfast.H)
	}
	// work: 8 hours = 48 rows
	if rectWork.H != 48 {
		t.Errorf("work: expected H=48, got H=%d", rectWork.H)
	}
}
