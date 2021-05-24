package controller

import (
	"container/list"
	"geck/driver"
	"geck/model"
	"geck/schedule"
	"log"
	"time"
)

type ZoneRun struct {
	StartTime time.Time
	Duration  time.Duration
	Zone      *Zone
}

// Zone
type Zone struct {
	actor   driver.WireActor
	weekSch schedule.WeeklySchedule
	locHist *list.List

	Schedule []schedule.Spec

	Id   string
	Name string
	Lane *Lane

	LastRun time.Time
	AccTime time.Duration

	version uint64
	enabled bool
}

type ZoneRequest struct {
	zone *Zone
	cb   chan *model.ZoneInfo
}

// Lane
type Lane struct {
	zones []*Zone
	Name  string

	runningZone *ZoneRun
	nextRun     *ZoneRun
	controller  *GardenController

	OobRunC  chan *ZoneRun
	OobStopC chan *Zone
	GetInfoC chan ZoneRequest
}

func getDuration(data interface{}) time.Duration {
	return data.(*model.ZoneScheduleSpec).Duration
}

func (zone * Zone) NextRun(t time.Time) *ZoneRun {
	lt, rt := zone.weekSch.GetNearest(t)

	if lt.Time.IsZero() || rt.Time.IsZero() {
		return nil
	}

	if zone.LastRun.Before(lt.Time) {
		return &ZoneRun{
			StartTime: t,
			Duration:  getDuration(lt.Data),
			Zone:      zone,
		}
	} else {
		return &ZoneRun{
			StartTime: rt.Time,
			Duration:  getDuration(rt.Data),
			Zone:      zone,
		}
	}
}

func (lane * Lane) nextActionIn(t time.Time) time.Duration {
	// Next default check
	timeout := time.Minute

	if lane.runningZone != nil {
		// Next finish. Make sure we end that.
		// We cannot do anything while something is running, so exit

		return lane.runningZone.StartTime.
			Add(lane.runningZone.Duration).
			Sub(t)
	}

	if lane.nextRun != nil {
		// Next start time
		tt := lane.nextRun.StartTime.Sub(t)

		if tt < timeout {
			timeout = tt
		}
	}


	return timeout
}

func (lane * Lane) LaneRouting() {
	lane.setNext(time.Now())

	for {
		timeout := lane.nextActionIn(time.Now())

		if timeout < 0 {
			lane.LaneTick(time.Now())
			continue
		}

		select {
		case <-time.After(timeout):
			lane.LaneTick(time.Now())

		case x := <-lane.OobRunC:
			lane.preempt(x, time.Now())

		case zone := <-lane.OobStopC:
			if zone.IsRunning() {
				log.Printf("Stop zone request: %s", zone.Id)
				lane.preempt(nil, time.Now())
			} else {
				log.Printf("Stop zone request ignored (not running): %s", zone.Id)
			}

		case req, ok := <-lane.GetInfoC:
			if !ok {
				return
			}

			req.cb <- req.zone.GetZoneInfo(time.Now())
		}
	}
}

func NewZone(lane *Lane, actor driver.WireActor, id string) *Zone {
	return &Zone{
		actor:    actor,
		weekSch:  schedule.WeeklySchedule{},
		Schedule: make([]schedule.Spec, 0),
		Id:       id,
		Lane:     lane,
		LastRun:  time.Now(),
		locHist:  list.New(),
	}
}

func NewLane(controller *GardenController, name string) *Lane {
	return &Lane{
		zones : []*Zone{},
		Name  : name,

		OobRunC:    make(chan *ZoneRun, 64),
		OobStopC:   make(chan *Zone, 64),
		GetInfoC:   make(chan ZoneRequest, 64),
		controller: controller,

		runningZone: nil,
		nextRun:     nil,
	}
}

func (zone * Zone) GetZoneInfo(t time.Time) *model.ZoneInfo {
	running := zone.Lane.runningZone

	zoneInfo := &model.ZoneInfo{
		ZoneInfoStatic: model.ZoneInfoStatic{
			Id:         zone.Id,
			Name:       zone.Name,
			Version:    zone.version,
			HardwareId: zone.actor.GetID(),
			Lane:       zone.Lane.Name,
			IsEnabled:  zone.enabled,
		},
		ZoneState:      model.ZoneState{
			IsRunning:  zone.IsRunning(),
			LastRun: 	zone.LastRun,
			Runtime:    zone.AccTime,
		},
	}

	nextRun := zone.NextRun(t)

	if nextRun != nil {
		zoneInfo.NextRun = &nextRun.StartTime
	}

	if running != nil && zoneInfo.IsRunning {
		zoneInfo.StartedAt = running.StartTime
	}

	zoneInfo.Schedule = make([]*model.ZoneScheduleSpec, len(zone.Schedule))

	for i, spec := range zone.Schedule {
		zoneInfo.Schedule[i] = &model.ZoneScheduleSpec{
			Idx:        i + 1,
			Duration:   getDuration(spec.Data),
			DaysOfWeek: spec.DaysOfWeek,
			Hours:      spec.Hours,
			Minutes:    spec.Minutes,
			AtTimeZone: spec.AtTimeZone,
		}
	}

	return zoneInfo
}


// Stop
func (zone *Zone) Stop() {
	zone.Lane.OobStopC <- zone
}

// Start
func (zone *Zone) Start(d time.Duration, t time.Time) {
	zone.Lane.OobRunC <- &ZoneRun{
		StartTime: t,
		Duration:  d,
		Zone:      zone,
	}
}

// internalStop
func (zone *Zone) internalStop(startTime time.Time, t time.Time) {
	zone.actor.Stop()

	run := &ZoneRun{
		StartTime: startTime,
		Duration:  t.Sub(startTime),
		Zone:      zone,
	}

	zone.AccTime += run.Duration

	log.Printf(
		"Zone finished: zone %s, run for %.2f minutes",
		zone.Id,
		run.Duration.Minutes())

	zone.locHist.PushFront(run)

	gc := zone.Lane.controller
	gc.historyC <- run

	zoneInfo := zone.GetZoneInfo(t)
	if err := gc.storage.UpdateZoneState(zoneInfo.Id, &zoneInfo.ZoneState); err != nil {
		log.Printf("Zone save error %s: %s", zone.Id, err.Error())
	}
}

// start the zone
func (run *ZoneRun) start(t time.Time) bool {
	zone := run.Zone

	if !zone.enabled {
		return false
	}

	zone.LastRun = t
	zone.actor.Start()
	zone.Lane.runningZone = run

	log.Printf("Starting: zone %s, at %s for %.2f minutes",
		zone.Id,
		run.StartTime.Format(time.RFC3339),
		run.Duration.Minutes())

	running := zone.actor.IsRunning()

	if !running {
		log.Printf("Failed to start: zone %s, disabling", zone.Id)
		zone.enabled = false
	}

	return running
}


// Update update schedule
func (zone *Zone) Update() {
	for _, sch := range zone.Schedule {
		err := sch.AddToSchedule(&zone.weekSch)

		if err != nil {
			log.Fatal(err)
		}
	}

	zone.version++
}

func (zone *Zone) IsRunning() bool {
	return zone.Lane.runningZone != nil && zone.Lane.runningZone.Zone == zone
}

// preempt add a
func (lane * Lane) preempt(run *ZoneRun, t time.Time) {
	active := lane.runningZone

	if active != nil {
		active.Zone.internalStop(active.StartTime, t)
		lane.runningZone = nil
		active.Duration = active.Duration - t.Sub(active.StartTime)
	}

	if run != nil {
		run.StartTime = t
		lane.nextRun = run
	} else {
		lane.setNext(t)
	}

	lane.LaneTick(t)
}

func (lane * Lane) setNext(t time.Time) {
	lane.nextRun = lane.findNext(t)

	if lane.nextRun != nil {
		log.Printf("Next run: zone %s, at %s for %.2f minutes",
			lane.nextRun.Zone.Id,
			lane.nextRun.StartTime.Format(time.RFC3339),
			lane.nextRun.Duration.Minutes())
	}
}

func (lane * Lane) findNext(t time.Time) *ZoneRun {
	var min *ZoneRun = nil

	for _, z := range lane.zones {
		if !z.enabled || z.IsRunning() || z.weekSch.Len() == 0 {
			continue
		}

		next := z.NextRun(t)

		if next != nil && (min == nil || next.StartTime.Before(min.StartTime)) {
			min = next
		}
	}

	return min
}

func (lane * Lane) LaneTick(t time.Time) {
	active := lane.runningZone

	if active != nil {
		if t.Sub(active.StartTime) < active.Duration {
			// Lane is still running
			return
		}

		active.Zone.internalStop(active.StartTime, t)
	}

	lane.runningZone = nil

	if lane.nextRun == nil {
		lane.setNext(t)
	}

	if lane.nextRun == nil {
		return
	}

	/**
	 * TODO : replace nextRun with heap, so that the
	 * previous next run is not lost when we replace it
	 */
	if !lane.nextRun.StartTime.After(t) {
		next := lane.nextRun
		next.StartTime = t
		next.start(t)
		lane.setNext(t)
	}
}
