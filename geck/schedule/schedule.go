package schedule

import (
	"fmt"
	"sort"
	"time"
)

type Entry interface {
	GetNearest(t time.Time) (time.Time, time.Time)
}

type Spec struct {
	DaysOfWeek []time.Weekday
	Hours      uint8
	Minutes    uint8
	AtTimeZone string
	Data	   interface{}
}

type slot struct {
	offset time.Duration
	data   interface{}
}

type TimeSlot struct {
	Time time.Time
	Data interface{}
}

type WeeklySchedule struct {
	loc   *time.Location
	times []slot // offsets from the beginning of Monday
}

func (w * WeeklySchedule) Len() int {
	return len(w.times)
}

func (w * WeeklySchedule) Less(i, j int) bool {
	return w.times[i].offset < w.times[j].offset
}

func (w * WeeklySchedule) Swap(i, j int) {
	w.times[i], w.times[j] = w.times[j], w.times[i]
}

func (w *WeeklySchedule) addSchedule(offsets []slot) {
	w.times = append(w.times, offsets...)
	sort.Sort(w)
}

func (w *WeeklySchedule) AddSpec(spec Spec) error {
	loc, err := time.LoadLocation(spec.AtTimeZone)
	if w.loc != nil && w.loc != loc {
		return fmt.Errorf("all timezones must be the same for one schedule")
	}

	w.loc = loc

	if err != nil {
		return err
	}

	if len(spec.DaysOfWeek) == 0 {
		return fmt.Errorf("no days specified")
	}

	times := make([]slot, len(spec.DaysOfWeek))
	tw := time.Hour * time.Duration(spec.Hours) + time.Minute * time.Duration(spec.Minutes)

	// Convert to absolute times from the beginning of the week
	for i, dow := range spec.DaysOfWeek {
		d := (int(dow) - int(time.Monday) + 7) % 7

		times[i] = slot{
			offset: time.Duration(d) * 24 * time.Hour + tw,
			data:   spec.Data,
		}
	}

	w.addSchedule(times)

	return nil
}

const week = time.Hour * 24 * 7

// GetCurrentWeekStart returns the time truncated
// to the beginning of the nearest Monday (to the left)
// with respect to the timezone
func GetCurrentWeekStart(t time.Time) time.Time {
	_, offset := t.Zone()
	truncTime := t.Truncate(week).Unix() - int64(offset)
	return time.Unix(truncTime, 0).In(t.Location())
}

func (s * slot) ToTimeSlot(base time.Time) TimeSlot {
	return TimeSlot{
		Time: base.Add(s.offset),
		Data: s.data,
	}
}

// GetNearest returns the nearest scheduled time to
// the left and nearest scheduled time to the right
func getNearestToArray(t time.Time, offsets []slot) (TimeSlot, TimeSlot) {
	start := GetCurrentWeekStart(t)

	i := sort.Search(len(offsets), func (i int) bool {
		return start.Add(offsets[i].offset).After(t)
	})

	var ltt, rtt TimeSlot

	if len(offsets) == 0 {
		return ltt, rtt
	}

	if i == 0 {
		ltt = offsets[len(offsets) - 1].ToTimeSlot(start.Add(-week))
	} else {
		ltt = offsets[i - 1].ToTimeSlot(start)
	}

	if i == len(offsets) {
		rtt = offsets[0].ToTimeSlot(start.Add(week))
	} else {
		rtt = offsets[i].ToTimeSlot(start)
	}

	return ltt, rtt
}

// GetNearest returns the nearest scheduled time to
// the left and nearest scheduled time to the right
func (w * WeeklySchedule) GetNearest(t time.Time) (TimeSlot, TimeSlot) {
	if w.loc == nil {
		w.loc = time.Local
	}

	return getNearestToArray(t.In(w.loc), w.times)
}
