package model

import "time"

/// ZoneRun a public representation of run history
type ZoneRun struct {
	Id       string 		`json:"id"`
	Started  time.Time      `json:"started"`
	Duration time.Duration  `json:"for"`
}

/// ZoneScheduleSpec a public representation of Schedule
type ZoneScheduleSpec struct {
	Idx        int            `json:"index"`
	Duration   time.Duration  `json:"for"`
	DaysOfWeek []time.Weekday `json:"days"`

	Hours      uint8  `json:"h"`
	Minutes    uint8  `json:"m"`
	AtTimeZone string `json:"tz"`
}

/// A structure for a public representation of zone static data
type ZoneInfoStatic struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Version	uint64 `json:"version"`

	IsEnabled  bool `json:"is_on"`
	HardwareId string `json:"hw_id"`
	Lane       string `json:"lane"`

	Schedule []*ZoneScheduleSpec `json:"schedule"`
}

/// A structure for a public representation of zone state
type ZoneState struct {
	IsRunning  bool `json:"is_running"`

	// Zone is disabled due to error
	Disabled   bool `json:"disabled"`

	NextRun    *time.Time    `json:"next_run"`
	StartedAt  time.Time     `json:"started_at"`
	LastRun    time.Time     `json:"last_run"`
	Runtime    time.Duration `json:"runtime"` // total zone run time
}

type ZoneInfo struct {
	ZoneInfoStatic
	ZoneState
}

/// StorageDriver - garden persistence engine
type StorageDriver interface {
	LoadZones() ([]*ZoneInfo, error)

	SaveZone(zone *ZoneInfoStatic) error
	UpdateZoneState(zoneId string, zone *ZoneState) error

	GetHistory(start time.Time, end time.Time) ([]ZoneRun, error)
	AddHistoryItem(ZoneRun * ZoneRun) error
}
