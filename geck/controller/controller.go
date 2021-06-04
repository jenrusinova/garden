package controller

import (
	"fmt"
	"geck/driver"
	"geck/model"
	"log"
	"sort"
	"sync/atomic"
	"time"
	"unsafe"
)

type Zone struct {
	info  *model.ZoneInfoStatic
	state ZoneStatePtr
	lane  *Lane
}

type GardenController struct {
	location *time.Location

	lanes map[string]*Lane
	zones map[string]*Zone

	historyC chan *ZoneRun

	driver    driver.WireDriver
	actorById map[string]driver.WireActor
	storage   model.StorageDriver
}

func NewGardenController(
	drv driver.WireDriver,
	storageDriver model.StorageDriver) *GardenController {
	gc := &GardenController{
		zones:     make(map[string]*Zone),
		lanes:     make(map[string]*Lane),
		historyC:  make(chan *ZoneRun, 64),
		driver:    drv,
		storage:   storageDriver,
		actorById: make(map[string]driver.WireActor),
		location:  time.Local,
	}

	return gc
}

// Stop
func (zone *Zone) Stop() {
	zone.lane.OobStopC <- ZoneIdType(zone.info.Id)
}

// Start
func (zone *Zone) Start(d time.Duration, t time.Time) {
	zone.lane.ScheduleC <- &ZoneRun{
		StartTime: t,
		Duration:  d,
		ZoneId:    ZoneIdType(zone.info.Id),
	}
}

func (gc * GardenController) ReloadZones() error {
	zones, err := gc.storage.LoadZones()
	if err != nil {
		panic(err)
	}

	byLane := make(map[string][]*model.ZoneInfo)
	actorsCheck := make(map[string]string)
	newZones := make(map[string]*Zone)

	for _, zone := range zones {
		_, ok := gc.actorById[zone.HardwareId]
		if !ok {
			return fmt.Errorf("hardware component not found: %s", zone.HardwareId)
		}

		if zone.Id == "" {
			return fmt.Errorf("zone does not have an id: %+v", zone)
		}

		byLane[zone.Lane] = append(byLane[zone.Lane], zone)

		if actorsCheck[zone.HardwareId] != "" {
			return fmt.Errorf("hardware pin %s is already assigned to zone %s",
				zone.HardwareId,
				actorsCheck[zone.HardwareId])
		}

		actorsCheck[zone.HardwareId] = zone.Id

		z := *zone // Copy of zone info
		newZones[zone.Id] = &Zone{
			info: &z.ZoneInfoStatic,
			state: makeZoneState(&z.ZoneState),
		}
	}

	newLanes := make(map[string]*Lane)
	oldLanes := gc.lanes

	for laneId, laneZones := range byLane {
		ln, found := oldLanes[laneId]

		if !found {
			ln = NewLane(gc, laneId)
			newLanes[laneId] = ln
			log.Printf("Starting lane %s", ln.Name)
			go ln.LaneController()
		}

		newLanes[ln.Name] = ln
		ln.ResetZones(laneZones)
	}

	for _, zone := range newZones {
		zone.lane = newLanes[zone.info.Lane]
	}

	// TODO: make atomic
	gc.lanes = newLanes
	gc.zones = newZones

	for laneId, oldLane := range oldLanes {
		if _, found := newLanes[laneId]; !found {
			oldLane.ResetZones(nil)
		}
	}

	return nil
}

func (gc *GardenController) ProcessHistory() {
	for history := range gc.historyC {
		_ = gc.storage.AddHistoryItem(&model.ZoneRun{
			Id:       string(history.ZoneId),
			Started:  history.StartTime,
			Duration: history.Duration,
		})
	}
}

// Run - main function for the controller
func (gc *GardenController) Startup() error {
	actors := gc.driver.AvailableActors()

	for _, actor := range actors {
		gc.actorById[actor.GetID()] = actor
	}

	err := gc.ReloadZones()
	if err != nil {
		log.Fatalf("Unable to load zones : %s", err.Error())
	}

	go gc.ProcessHistory()

	return nil
}

// StartZone start zone
func (gc * GardenController) StartZone(
	id string,
	duration time.Duration,
	preemptive bool) error {
	zone, ok := gc.zones[id]

	if !ok {
		return fmt.Errorf("zone not found : %s", id)
	}

	zone.Start(duration, time.Now())
	return nil
}

// StopZone stop zone
func (gc * GardenController) StopZone(id string) error {
	zone, ok := gc.zones[id]

	if !ok {
		return fmt.Errorf("zone not found : %s", id)
	}

	zone.Stop()
	return nil
}

func (gc *GardenController) ZoneFinish(run ZoneRun) {
	gc.historyC <- &run
}

func (gc *GardenController) UpdateZoneState(id ZoneIdType, state model.ZoneState) {
	gc.zones[string(id)].state.Set(&state)

	if err := gc.storage.UpdateZoneState(string(id), &state); err != nil {
		log.Printf("Zone save error %s: %s", id, err.Error())
	}
}

func (gc *GardenController) Shutdown() {
	for _, lane := range gc.lanes {
		close(lane.ResetC)
	}

	close(gc.historyC)
}

// GetZoneInfo race condition safe get info for a zone
func (gc *GardenController) GetZoneInfo(zoneId string) []*model.ZoneInfo {
	zones := gc.zones

	if zoneId != "" {
		zone, ok := gc.zones[zoneId]
		if !ok {
			return nil
		}

		return []*model.ZoneInfo{
			{
				ZoneInfoStatic: *zone.info,
				ZoneState:      *zone.state.Get(),
			},
		}
	}

	result := make([]*model.ZoneInfo, 0, len(zones))

	for _, zone := range zones {
		result = append(result, &model.ZoneInfo{
			ZoneInfoStatic: *zone.info,
			ZoneState:      *zone.state.Get(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]

		if a.Lane != b.Lane {
			return a.Lane < b.Lane
		}

		return a.Id < b.Id
	})

	return result
}

func (gc *GardenController) UpdateZone(zone *model.ZoneInfoStatic, resetEnabled bool) error {
	lookupZone := gc.GetZoneInfo(zone.Id)
	if len(lookupZone) > 1 {
		return fmt.Errorf("incorrect id in request: %+v", zone)
	}

	createNewZone := len(lookupZone) == 0

	if createNewZone {
		if err := gc.validateZone(zone); err != nil {
			return err
		}
	} else {
		// Edit existing zone
		existingZone := lookupZone[0]

		if zone.HardwareId != "" {
			existingZone.HardwareId = zone.HardwareId
		}

		if zone.Name != "" {
			existingZone.Name = zone.Name
		}

		if resetEnabled {
			existingZone.IsEnabled = zone.IsEnabled
		}

		if zone.Schedule != nil {
			existingZone.Schedule = zone.Schedule
		}

		if zone.Lane != "" {
			existingZone.Lane = zone.Lane
		}

		zone = &existingZone.ZoneInfoStatic
	}

	if err := gc.storage.SaveZone(zone); err != nil {
		return err
	}

	if err := gc.ReloadZones(); err != nil {
		return err
	}

	return nil
}

func (gc *GardenController) validateZone(zone *model.ZoneInfoStatic) error {
	if zone.Id == "" {
		return fmt.Errorf("id not set: %+v", *zone)
	}

	if zone.Name == "" {
		return fmt.Errorf("name not set: %+v", *zone)
	}

	if _, found := gc.actorById[zone.HardwareId]; !found {
		return fmt.Errorf("hardware element not found: %+v", *zone)
	}

	return nil
}

type ZoneStatePtr struct {
	state unsafe.Pointer
}

func makeZoneState(state *model.ZoneState) ZoneStatePtr {
	return ZoneStatePtr{
		state: unsafe.Pointer(state),
	}
}

func (zptr *ZoneStatePtr) Set(state *model.ZoneState) {
	atomic.StorePointer(&zptr.state, unsafe.Pointer(state))
}

func (zptr *ZoneStatePtr) Get() *model.ZoneState {
	return (*model.ZoneState)(atomic.LoadPointer(&zptr.state))
}
