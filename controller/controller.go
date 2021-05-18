package controller

import (
	"fmt"
	"geck/driver"
	"geck/model"
	"geck/schedule"
	"log"
	"time"
)

type GardenController struct {
	zones []*Zone

	location *time.Location

	laneById map[string]*Lane
	zoneById map[string]*Zone

	historyC chan *ZoneRun

	driver    driver.WireDriver
	actorById map[string]driver.WireActor
	storage   model.StorageDriver
}


func NewGardenController(
	drv driver.WireDriver,
	storageDriver model.StorageDriver) *GardenController {
	gc := &GardenController{
		zones:     nil,
		zoneById:  make(map[string]*Zone),
		laneById:  make(map[string]*Lane),
		historyC:  make(chan *ZoneRun, 64),
		driver:    drv,
		storage:   storageDriver,
		actorById: make(map[string]driver.WireActor),
		location:  time.Local,
	}

	return gc
}

// Run - main function for the controller
func (gc * GardenController) Startup() error {
	actors := gc.driver.AvailableActors()

	for _, actor := range actors {
		gc.actorById[actor.GetID()] = actor
	}

	zones, err := gc.storage.LoadZones()

	if err != nil {
		panic(err)
	}

	for _, zone := range zones {
		gc.InitZone(zone)
	}

	for name, lane := range gc.laneById {
		log.Printf("Starting lane %s", name)
		go lane.LaneRouting()
	}

	go func() {
		for {
			select {
			case history, ok := <-gc.historyC:
				if !ok {
					return
				}

				_ = gc.storage.AddHistoryItem(&model.ZoneRun{
					Id:       history.Zone.Id,
					Started:  history.StartTime,
					Duration: history.Duration,
				})
			}
		}
	}()

	return nil
}

// StartZone start zone
func (gc * GardenController) StartZone(
	id string,
	duration time.Duration,
	preemptive bool) error {
	zone, ok := gc.zoneById[id]

	if !ok {
		return fmt.Errorf("zone not found : %s", id)
	}

	zone.Start(duration, time.Now())
	return nil
}

// StopZone stop zone
func (gc * GardenController) StopZone(id string) error {
	zone, ok := gc.zoneById[id]

	if !ok {
		return fmt.Errorf("zone not found : %s", id)
	}

	zone.Stop()
	return nil
}

func (zone * Zone) AssignSchedule(spec * model.ZoneScheduleSpec) {
	var item *schedule.Spec

	if spec.Idx == 0 || spec.Idx - 1 == len(zone.Schedule) {
		zone.Schedule = append(zone.Schedule, schedule.Spec{})
		spec.Idx = len(zone.Schedule)
	}

	item = &zone.Schedule[spec.Idx - 1]
	item.Data = spec
	item.Hours = spec.Hours
	item.Minutes = spec.Minutes
	item.AtTimeZone = spec.AtTimeZone
	item.DaysOfWeek = spec.DaysOfWeek
}


func (gc * GardenController) InitZone(zoneInfo *model.ZoneInfo) {
	lane := gc.GetLane(zoneInfo.Lane)
	hw, ok := gc.actorById[zoneInfo.HardwareId]

	if !ok {
		log.Fatalf("Hardware component not found : %s", zoneInfo.HardwareId)
	}

	zone := NewZone(lane, hw, zoneInfo.Id)
	lane.zones = append(lane.zones, zone)
	gc.zones = append(gc.zones, zone)

	if _, ok := gc.zoneById[zone.Id]; ok {
		log.Fatalf("Zone already exists : %s", zoneInfo.Id)
	}

	gc.zoneById[zone.Id] = zone

	zone.Name = zoneInfo.Name
	zone.enabled = zoneInfo.IsEnabled

	if !zone.LastRun.IsZero() {
		zone.LastRun = zoneInfo.LastRun
	}

	zone.AccTime = zoneInfo.Runtime

	for _, sch := range zoneInfo.Schedule {
		if sch.AtTimeZone == "" {
			sch.AtTimeZone = gc.location.String()
		}

		zone.AssignSchedule(sch)
	}

	zone.Update()
}


func (gc * GardenController) GetLane(laneName string) * Lane {
	result, ok := gc.laneById[laneName]

	if !ok {
		result = NewLane(gc, laneName)
		gc.laneById[laneName] = result
	}

	return result
}


func (gc * GardenController) Shutdown() {
	for _, lane := range gc.laneById {
		close(lane.GetInfoC)
	}

	close(gc.historyC)
}

// GetZoneInfo race condition safe get info for a zone
func (gc * GardenController) GetZoneInfo(zoneId string) []*model.ZoneInfo {
	var zonesToFind []*Zone

	if zoneId == "" {
		zonesToFind = make([]*Zone, len(gc.zones))

		for i, zone := range gc.zones {
			zonesToFind[i] = zone
		}
	} else {
		zonesToFind = make([]*Zone, 1)
		zonesToFind[0] = gc.zoneById[zoneId]
	}

	result := make([]*model.ZoneInfo, len(zonesToFind))
	returnChannel := make(chan *model.ZoneInfo, len(zonesToFind))

	for _, zone := range zonesToFind {
		zone.Lane.GetInfoC <- ZoneRequest{
			zone: zone,
			cb:   returnChannel,
		}
	}

	for i, _ := range zonesToFind {
		result[i] = <-returnChannel
	}

	return result
}
