package controller

import (
	"geck/driver"
	"geck/model"
	"geck/schedule"
	"log"
	"time"
)

type ZoneIdType string

type ZoneRun struct {
	StartTime time.Time
	Duration  time.Duration
	ZoneId    ZoneIdType
}

type ZoneRunData struct {
	ZoneRun

	// We always store the hardware pin of the running zone,
	// to be able to stop it even if it's deleted
	actor driver.WireActor
}

type ZoneRuntimeState struct {
	Id      ZoneIdType

	State   model.ZoneState
	enabled bool
	actor   driver.WireActor
	weekSch schedule.WeeklySchedule
}

// Lane
type Lane struct {
	Name   string
	zones  map[ZoneIdType]*ZoneRuntimeState
	hwPins map[string]driver.WireActor

	// currently running zone
	runningZone *ZoneRunData

	// cached next run, which we are waiting for
	nextRun *ZoneRunData

	OobStopC  chan ZoneIdType
	ScheduleC chan *ZoneRun

	// Channel to update the whole zone info
	ResetC chan []*model.ZoneInfo

	// callbacks for upper level
	OnZoneFinish func(ZoneRun)
	UpdateZoneState func(ZoneIdType, model.ZoneState)
}

func (lane *Lane) Shutdown() {
	close(lane.ResetC)
}

func getDuration(data interface{}) time.Duration {
	return data.(*model.ZoneScheduleSpec).Duration
}

func (lane *Lane) NextZoneRun(zone *ZoneRuntimeState, t time.Time) *ZoneRunData {
	// We cannot just pass last run here as t because
	// we want the next run to happen right after the current time

	lt, rt := zone.weekSch.GetNearest(t)

	if lt.Time.IsZero() || rt.Time.IsZero() {
		return nil
	}

	result := &ZoneRunData{
		ZoneRun: ZoneRun{
			StartTime: rt.Time,
			Duration:  getDuration(rt.Data),
			ZoneId:    zone.Id,
		},
		actor: zone.actor,
	}

	if zone.State.LastRun.Before(lt.Time) {
		// We skipped one or more runs, so schedule one immediately
		result.StartTime = t
		result.Duration = getDuration(lt.Data)
	}

	if zone.State.NextRun == nil || !zone.State.NextRun.Equal(result.StartTime) {
		tt := result.StartTime
		zone.State.NextRun = &tt
		lane.UpdateZoneState(zone.Id, zone.State)
	}

	return result
}

func (lane *Lane) nextActionIn(t time.Time) time.Duration {
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

func (lane *Lane) LaneController() {
	lane.setNext(time.Now())

	for {
		timeout := lane.nextActionIn(time.Now())

		if timeout < 0 {
			lane.LaneTick(time.Now())
			continue
		}

		select {
		case <-time.After(timeout):
		case x := <-lane.ScheduleC:
			lane.preempt(x, time.Now())
		case zone := <-lane.OobStopC:
			if lane.runningZone != nil && zone == lane.runningZone.ZoneId {
				log.Printf("Stop zone request: %s", zone)
				lane.preempt(nil, time.Now())
			} else {
				log.Printf("Stop zone request ignored (not running): %s", zone)
			}

		case newZones, ok := <-lane.ResetC:
			if !ok || newZones == nil {
				lane.stopZone(time.Now())
				return
			}

			lane.reset(newZones)
		}

		lane.LaneTick(time.Now())
	}
}

func (lane *Lane) reset(zones []*model.ZoneInfo) {
	lane.zones = make(map[ZoneIdType]*ZoneRuntimeState)

	for _, zone := range zones {
		zoneRun := makeZoneRunState(
			lane.hwPins[zone.HardwareId],
			zone)

		lane.zones[zoneRun.Id] = zoneRun
	}

	lane.setNext(time.Now())

	for id, zoneRun := range lane.zones {
		// update state
		wasInRunningState := zoneRun.State.IsRunning
		zoneRun.State.IsRunning = zoneRun.actor.IsRunning()
		zoneRun.State.Disabled = !zoneRun.enabled

		if wasInRunningState != zoneRun.State.IsRunning {
			lane.UpdateZoneState(id, zoneRun.State)
		}
	}
}

func makeZoneRunState(actor driver.WireActor, zoneInfo *model.ZoneInfo) *ZoneRuntimeState {
	return &ZoneRuntimeState{
		Id:      ZoneIdType(zoneInfo.Id),
		State:   zoneInfo.ZoneState, // copy

		// We only rely on static state for now, so every
		//  change will reset it && !zoneInfo.Disabled,
		enabled: zoneInfo.IsEnabled,
		actor:   actor,
		weekSch: schedule.WeeklySchedule{},
	}
}

func NewLane(gc *GardenController, name string) *Lane {
	return &Lane{
		Name  : name,
		hwPins : gc.actorById,

		ScheduleC: make(chan *ZoneRun, 64),
		OobStopC:  make(chan ZoneIdType, 64),
		ResetC:    make(chan []*model.ZoneInfo, 64),

		runningZone: nil,
		nextRun:     nil,

		OnZoneFinish: gc.ZoneFinish,
		UpdateZoneState: gc.UpdateZoneState,
	}
}

// internalStop
func (lane *Lane) stopZone(stopTime time.Time) {
	zone := lane.runningZone
	if zone == nil {
		return
	}

	zone.actor.Stop()
	lane.runningZone = nil

	run := zone.ZoneRun
	run.Duration = stopTime.Sub(run.StartTime)

	lane.OnZoneFinish(run)

	zoneData, ok := lane.zones[run.ZoneId]

	if !ok {
		// Zone was removed from the lane
		return
	}

	zoneData.State.Runtime += run.Duration
	zoneData.State.IsRunning = false

	log.Printf(
		"Zone finished: zone %s, run for %.2f minutes",
		zone.ZoneId,
		run.Duration.Minutes())

	lane.UpdateZoneState(run.ZoneId, zoneData.State)
}

// start the zone
func (lane *Lane) startZone(run *ZoneRunData, t time.Time) bool {
	zone, ok := lane.zones[run.ZoneId]

	if !ok || !zone.enabled {
		return false
	}

	zone.State.LastRun = t
	zone.actor.Start()
	lane.runningZone = run

	log.Printf("Starting: zone %s, at %s for %.2f minutes",
		zone.Id,
		run.StartTime.Format(time.RFC3339),
		run.Duration.Minutes())

	running := zone.actor.IsRunning()
	zone.State.IsRunning = running

	if !running {
		log.Printf("Failed to start: zone %s, disabling", zone.Id)
		zone.enabled = false
		zone.State.Disabled = true
	}

	lane.UpdateZoneState(zone.Id, zone.State)

	return running

}

// Update update schedule
func (zone *ZoneRuntimeState) UpdateSchedule(specs []model.ZoneScheduleSpec, defaultLocation string) {
	zone.weekSch = schedule.WeeklySchedule{}

	for _, sch := range specs {
		spec := schedule.Spec{
			DaysOfWeek: sch.DaysOfWeek,
			Hours:      sch.Hours,
			Minutes:    sch.Minutes,
			AtTimeZone: sch.AtTimeZone,
			Data:       &sch,
		}

		if spec.AtTimeZone == "" {
			spec.AtTimeZone = defaultLocation
		}

		err := zone.weekSch.AddSpec(spec)
		if err != nil {
			log.Printf("Cannot build schedule from spec %+v : %s", spec, err.Error())
		}
	}
}

// preempt add a
func (lane *Lane) preempt(run *ZoneRun, t time.Time) {
	active := lane.runningZone

	if active != nil {
		lane.stopZone(t)
		// TODO: Maybe schedule leftovers of this run for later?
	}

	if run == nil {
		lane.setNext(t)
		return
	}

	run.StartTime = t
	zone := lane.zones[run.ZoneId]

	if zone != nil {
		lane.nextRun = &ZoneRunData{
			ZoneRun: *run,
			actor:   zone.actor,
		}
	}
}

func (lane * Lane) setNext(t time.Time) {
	lane.nextRun = lane.findNext(t)

	if lane.nextRun == nil {
		return
	}

	log.Printf("Next run: zone %s, at %s for %.2f minutes",
		lane.nextRun.ZoneId,
		lane.nextRun.StartTime.Format(time.RFC3339),
		lane.nextRun.Duration.Minutes())
}

func (lane * Lane) findNext(t time.Time) *ZoneRunData {
	var min *ZoneRunData = nil

	for _, z := range lane.zones {
		if !z.enabled || z.State.IsRunning || z.weekSch.Len() == 0 {
			continue
		}

		next := lane.NextZoneRun(z, t)

		if next != nil && (min == nil || next.StartTime.Before(min.StartTime)) {
			min = next
		}
	}

	return min
}

func (lane * Lane) LaneTick(t time.Time) bool {
	active := lane.runningZone

	if active != nil {
		if t.Sub(active.StartTime) < active.Duration {
			// Lane is still running
			return true
		}

		lane.stopZone(t)
	}

	lane.runningZone = nil

	if lane.nextRun == nil {
		lane.setNext(t)
	}

	if lane.nextRun == nil {
		// End lane, nothing to run
		return false
	}

	/**
	 * TODO : replace nextRun with heap, so that the
	 *  previous next run is not lost when we replace it
	 */
	if !lane.nextRun.StartTime.After(t) {
		next := lane.nextRun
		next.StartTime = t
		lane.startZone(next, t)
		lane.setNext(t)
	}

	return true
}

func (lane *Lane) ResetZones(zones []*model.ZoneInfo) {
	lane.ResetC <- zones
}
