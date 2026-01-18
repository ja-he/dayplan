package cli

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/model"
)

func (c *Controller) RemoveCurrentEvent() {
	c.removeEvent(c.data.CurrentEventID)
}

func (c *Controller) switchToNextEventInDay() {
	defer c.ensureCurrentEventVisible()

	if c.data.CurrentEventID == nil {
		candidate, err := c.eventsProvider.GetEventAfter(c.data.CurrentDate.ToGotime(time.Local))
		if err != nil {
			c.log.Error().Err(err).Stringer("date", c.data.CurrentDate).Msg("could not get next for current date")
			return
		}
		if candidate == nil {
			c.log.Warn().Msgf("No model after exists.")
			return
		}
		if model.DateFromGotime(candidate.Start, time.Local) == c.data.CurrentDate {
			c.data.CurrentEventID = new(model.EventID)
			*c.data.CurrentEventID = candidate.ID
			c.log.Debug().Stringer("event", candidate).Msg("switched to next event")
			return
		}
		c.log.Debug().Msg("no event on current day")
		return
	}

	currentEventID := *c.data.CurrentEventID
	{
		e, err := c.eventsProvider.GetEvent(currentEventID)
		if err != nil {
			c.log.Error().
				Str("ID", currentEventID).
				Msg("Could not find current event to check validity, will set to nil.")
			c.data.CurrentEventID = nil
			return
		}
		currentDate := c.data.CurrentDate
		eStartDate := model.DateFromGotime(e.Start, time.Local)
		if currentDate != eStartDate {
			c.log.Error().
				Stringer("currentDate", currentDate).
				Stringer("eStartDate", eStartDate).
				Str("ID", currentEventID).
				Msg("Current event is not on current date.")
		}
	}

	next, err := c.eventsProvider.GetFollowingEvent(currentEventID)
	if err != nil {
		c.log.Error().Err(err).Str("id", string(currentEventID)).Msg("could not get following event of current event")
		return
	}
	if next == nil {
		c.log.Info().Msg("there is no next event")
		return
	}
	if model.DateFromGotime(next.Start, time.Local) != c.data.CurrentDate {
		c.log.Info().Msg("next event is on a different day")
		return
	}

	c.data.CurrentEventID = new(model.EventID)
	*c.data.CurrentEventID = next.ID
	c.log.Debug().Stringer("event", next).Msg("switched to next event")
}

func (c *Controller) switchToPreviousEventInDay() {
	defer c.ensureCurrentEventVisible()

	if c.data.CurrentEventID == nil {
		candidate, err := c.eventsProvider.GetEventBefore(c.data.CurrentDate.ToGotime(time.Local).Add(24 * time.Hour))
		if err != nil {
			c.log.Error().Err(err).Stringer("date", c.data.CurrentDate).Msg("could not get prev for current date")
			return
		}
		if candidate == nil {
			c.log.Warn().Msgf("No model before exists.")
			return
		}
		if model.DateFromGotime(candidate.Start, time.Local) == c.data.CurrentDate {
			c.data.CurrentEventID = new(model.EventID)
			*c.data.CurrentEventID = candidate.ID
			c.log.Debug().Stringer("event", candidate).Msg("switched to prev event")
			return
		}
		c.log.Debug().Msg("no event on current day")
		return
	}

	prev, err := c.eventsProvider.GetPrecedingEvent(*c.data.CurrentEventID)
	if err != nil {
		c.log.Error().Err(err).Stringer("date", c.data.CurrentDate).Msg("could not get prev for current date")
		return
	}
	if prev == nil {
		c.log.Info().Msg("there is no prev event")
		return
	}
	if model.DateFromGotime(prev.Start, time.Local) != c.data.CurrentDate {
		c.log.Info().Msg("prev event is on a different day")
		return
	}

	// current event ID is not nil, so we can just set it to the previous event's
	if prev.ID == *c.data.CurrentEventID {
		log.Error().Msgf("Got same event ID back for previous event ('%s')", *c.data.CurrentEventID)
		return
	}
	*c.data.CurrentEventID = prev.ID
	c.log.Debug().Stringer("event", prev).Msg("switched to prev event")
}

func (c *Controller) getCurrentDayEvents() ([]*model.Event, error) {
	startTime := c.data.CurrentDate.ToGotime(time.Local)
	endTime := startTime.Add(24 * time.Hour)
	events, err := c.eventsProvider.GetEventsCoveringTimerange(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("could not get events for current day (%w)", err)
	}
	return events, nil
}
func (c *Controller) ensureCurrentEventVisible() {
	id := c.data.CurrentEventID
	if id == nil {
		c.log.Info().Msg("no current event selected, so nothing to ensure visible")
		return
	}
	e, err := c.eventsProvider.GetEvent(*id)
	if err != nil {
		c.log.Error().Err(err).Msg("could not get current event while ensuring visibility")
		return
	}
	c.ensureEventsPaneTimestampWithinVisibleScroll(e.Start)
	c.ensureEventsPaneTimestampWithinVisibleScroll(e.End)
}

func (c *Controller) moveEventsForwardPushing() error {
	pushDuration := c.data.MainTimelineViewParams.DurationOfHeight(1)
	pushResolution := c.data.MainTimelineViewParams.DurationOfHeight(1)
	return c.moveEventsPushingBy(pushDuration, pushResolution)
}

func (c *Controller) moveEventsBackwardPushing() error {
	pushDuration := -c.data.MainTimelineViewParams.DurationOfHeight(1)
	pushResolution := c.data.MainTimelineViewParams.DurationOfHeight(1)
	return c.moveEventsPushingBy(pushDuration, pushResolution)
}

// moves events pushing other events
//
// d this is how "far" everything gets pushed.
//
// m is basically the "modulus", i.e. if something needs to get snapped to the
// grid of visible resolution, this is that, not sure yet if needed really.
func (c *Controller) moveEventsPushingBy(d, m time.Duration) error {
	return fmt.Errorf("unimplemented (this should push for %s (with res %s))", d, m)
}

func (c *Controller) removeEvent(id *model.EventID) {
	if id == nil {
		c.log.Trace().Msg("removeEvent given nil, returning without action.")
		return
	}
	c.log.Trace().Msgf("removing event '%s'.", *id)

	isCurrentEvent := c.data.CurrentEventID != nil && *c.data.CurrentEventID == *id
	var newCurrentEventID *model.EventID
	if isCurrentEvent {
		nextEvent, err := c.eventsProvider.GetFollowingEvent(*id)
		if err != nil {
			c.log.Error().Err(err).Msg("could not get following event")
		} else if nextEvent == nil || !c.data.CurrentDate.Is(nextEvent.Start, time.Local) {
			prevEvent, err := c.eventsProvider.GetPrecedingEvent(*id)
			if err != nil {
				c.log.Error().Err(err).Msg("could not get preceding event")
			} else if nextEvent != nil && c.data.CurrentDate.Is(prevEvent.End, time.Local) {
				newCurrentEventID = new(model.EventID)
				*newCurrentEventID = prevEvent.ID
				c.log.Debug().Msgf("will switch to previous event: %s", *newCurrentEventID)
			}
		} else {
			newCurrentEventID = new(model.EventID)
			*newCurrentEventID = nextEvent.ID
			c.log.Debug().Msgf("will switch to next event: %s", *newCurrentEventID)
		}
		if newCurrentEventID == nil {
			c.log.Debug().Msg("no next/prev event to switch to")
		}
	}
	err := c.eventsProvider.RemoveEvent(*id)
	if err != nil {
		c.log.Error().Err(err).Msg("could not remove event")
		return
	}
	if isCurrentEvent {
		if newCurrentEventID != nil {
			c.log.Debug().Msgf("updating current event to %s", *newCurrentEventID)
			c.data.CurrentEventID = new(model.EventID)
			*c.data.CurrentEventID = *newCurrentEventID
			c.ensureCurrentEventVisible()
		} else {
			c.log.Debug().Msg("nilling current event")
			c.data.CurrentEventID = nil
		}
	}
}
