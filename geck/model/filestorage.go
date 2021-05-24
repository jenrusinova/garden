package model

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"time"
)


type staticConfigFile struct {
	Zones []*ZoneInfoStatic `json:"zones"`
}

type getZonesContext struct {
}


type saveZoneContext struct {
	zone *ZoneInfoStatic
}

type updateZoneStateContext struct {
	zoneId string
	state *ZoneState
}

type getHistoryContext struct {
	start  time.Time
	end    time.Time
}


type addHistoryContext struct {
	history *ZoneRun
}


type QueryContextBase struct {
	ctx     context.Context
	query   interface{}

	// To avoid boilerplate code
	result  chan interface{}
	err 	error
}


type DirectoryStorageDriver struct {
	FilePath string

	zoneMap  map[string]*ZoneInfoStatic
	queriesC chan *QueryContextBase
	running  int32

	zoneStaticConfig staticConfigFile
}

var _ StorageDriver = &DirectoryStorageDriver{}

const zoneStaticFile = "zones.conf.json"

func (fsd *DirectoryStorageDriver) saveJsonToFile(entity interface{}, file string) error {
	data, err := json.MarshalIndent(entity, "", " ")

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(fsd.FilePath, file), data, 0644)

	if err != nil {
		return err
	}

	return nil
}

func (fsd *DirectoryStorageDriver) doQuery(query interface{}) (interface{}, error) {
	queryWithContext := QueryContextBase{

		// Currently ignore this context all together
		ctx:    context.Background(),
		query:  query,
		result: make(chan interface{}),
		err:    nil,
	}

	fsd.queriesC <- &queryWithContext

	select {
	case <- queryWithContext.ctx.Done():
		return nil, queryWithContext.ctx.Err()
	case result, ok := <- queryWithContext.result:
		if !ok {
			return nil, queryWithContext.err
		}

		return result, nil
	}

}

// NewDirectoryStorageDriver create a single file database storage driver
func NewDirectoryStorageDriver(FileName string) *DirectoryStorageDriver {
	return &DirectoryStorageDriver{
		FilePath:         FileName,
		zoneStaticConfig: staticConfigFile{},
		queriesC:         make(chan *QueryContextBase, 16),
		zoneMap:          make(map[string]*ZoneInfoStatic),
	}
}


func (fsd *DirectoryStorageDriver) ProcessRequest(request *QueryContextBase) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Internal error : %+v", err)
			request.err = fmt.Errorf("internal error while processing request")
		}

		close(request.result)
	}()

	select {
	case <-request.ctx.Done():
		request.err = fmt.Errorf("request cancelled")
		return
	default:
	}

	switch query := request.query.(type) {
	case getHistoryContext:
		var result []*ZoneRun

		if result, request.err = fsd.doGetHistory(query); request.err == nil {
			request.result <- result
		}
	case addHistoryContext:
		if request.err = fsd.doAddHistoryItem(query); request.err == nil {
			request.result <- struct{}{}
		}
	case getZonesContext:
		var result []*ZoneInfo

		if result, request.err = fsd.doLoadZones(); request.err == nil {
			request.result <- result
		}
	case saveZoneContext:
		if request.err = fsd.doSaveZone(query); request.err == nil {
			request.result <- struct{}{}
		}
	case updateZoneStateContext:
		if request.err = fsd.doUpdateZoneState(query); request.err == nil {
			request.result <- struct{}{}
		}
	}
}

func (fsd *DirectoryStorageDriver) Run() {
	for request := range fsd.queriesC {
		fsd.ProcessRequest(request)
	}
}


func (fsd *DirectoryStorageDriver) Shutdown() {
	close(fsd.queriesC)
}


func (fsd *DirectoryStorageDriver) Startup() error {
	if err := fsd.loadFromFile()
		err != nil {
		return err
	}

	go fsd.Run()
	return nil
}

/// SaveZone public thread safe interface for updating zone info
func (fsd *DirectoryStorageDriver) SaveZone(zone *ZoneInfoStatic) error {
	if _, err := fsd.doQuery(saveZoneContext{zone: zone}); err != nil {
		return fmt.Errorf("request error : %s", err.Error())
	}

	return nil
}

/// doSaveZone actual implementation for updating zone info
func (fsd *DirectoryStorageDriver) doSaveZone(ctx saveZoneContext) error {
	zone := ctx.zone
	zonePtr, ok := fsd.zoneMap[zone.Id]

	if !ok {
		zonePtr = &ZoneInfoStatic{}
		fsd.zoneStaticConfig.Zones = append(fsd.zoneStaticConfig.Zones, zonePtr)
		fsd.zoneMap[zone.Id] = zonePtr
	}

	*zonePtr = *zone
	return fsd.saveJsonToFile(fsd.zoneStaticConfig, zoneStaticFile)
}

func (fsd *DirectoryStorageDriver) UpdateZoneState(zoneId string, zone *ZoneState) error {
	if _, err := fsd.doQuery(updateZoneStateContext{zoneId: zoneId, state: zone}); err != nil {
		return fmt.Errorf("request error : %s", err.Error())
	}

	return nil
}

/// doUpdateZoneState actual implementation for updating zone info
func (fsd *DirectoryStorageDriver) doUpdateZoneState(ctx updateZoneStateContext) error {
	err := fsd.saveJsonToFile(ctx.state, "_zone_" + ctx.zoneId + ".json")

	if err != nil {
		return err
	}

	return nil
}

/// LoadZones load or reload zone information from storage
func (fsd *DirectoryStorageDriver) LoadZones() ([]*ZoneInfo, error) {
	result, err := fsd.doQuery(getZonesContext{})

	if err != nil {
		return nil, fmt.Errorf("request error : %s", err.Error())
	}

	if castResult, ok := result.([]*ZoneInfo); ok {
		return castResult, nil
	}

	return nil, fmt.Errorf("invalid response : %+v", result)
}

/// doLoadZones implements load or reload zone information from storage
func (fsd *DirectoryStorageDriver) doLoadZones() ([]*ZoneInfo, error) {
	if err := fsd.loadFromFile(); err != nil {
		return nil, err
	}

	zoneResult := make([]*ZoneInfo, len(fsd.zoneStaticConfig.Zones))

	for i, staticInfo := range fsd.zoneStaticConfig.Zones {
		data, err := ioutil.ReadFile(path.Join(fsd.FilePath, "_zone_" + staticInfo.Id + ".json"))

		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		zoneResult[i] = &ZoneInfo{
			ZoneInfoStatic: *staticInfo,
		}

		if data != nil {
			if err = json.Unmarshal(data, &zoneResult[i].ZoneState); err != nil {
				return nil, err
			}
		}
	}

	return zoneResult, nil
}

func (fsd *DirectoryStorageDriver) GetHistory(start time.Time, end time.Time) ([]ZoneRun, error) {
	result, err := fsd.doQuery(getHistoryContext{
		start:  start,
		end:    end,
	})

	if err != nil {
		return nil, fmt.Errorf("request error : %s", err.Error())
	}

	if castResult, ok := result.([]ZoneRun); ok {
		return castResult, nil
	}

	return nil, fmt.Errorf("invalid response : %+v", result)
}


func (fsd *DirectoryStorageDriver) doGetHistory(ctx getHistoryContext) ([]*ZoneRun, error) {
	f, err := os.Open(path.Join(fsd.FilePath, "history.csv"))

	if err != nil {
		return nil, err
	}

	defer f.Close()

	reader := csv.NewReader(f)
	reader.ReuseRecord = true

	result := make([]*ZoneRun, 0, 1024)

	for {
		x, err := reader.Read()

		if err == io.EOF {
			return result, nil
		}

		if err != nil {
			return nil, err
		}

		record := ZoneRun{ Id: x[0] }

		if err = record.Started.UnmarshalText([]byte(x[1])); err != nil {
			return nil, err
		}

		dur, err := strconv.ParseInt(x[2], 10, 64);

		if err != nil {
			return nil, err
		}

		record.Duration = time.Duration(dur)

		if record.Started.After(ctx.start) && record.Started.Before(ctx.end) {
			result = append(result, &record)
		}
	}
}

func (fsd *DirectoryStorageDriver) AddHistoryItem(zoneRun *ZoneRun) error {
	_, err := fsd.doQuery(addHistoryContext{
		history: zoneRun,
	})

	return err
}


func (fsd *DirectoryStorageDriver) doAddHistoryItem(ctx addHistoryContext) error {
	f, err := os.OpenFile(
		path.Join(fsd.FilePath, "history.csv"),
		os.O_APPEND|os.O_CREATE,
		0666,
	)

	if err != nil {
		return err
	}

	defer f.Close()

	zone := ctx.history
	startTime, _ := zone.Started.MarshalText()

	err = csv.
		NewWriter(f).
		Write([]string{
			zone.Id,
			string(startTime),
			strconv.FormatInt(int64(zone.Duration), 10)})

	if err != nil {
		return err
	}

	err = f.Sync()

	return err
}

func (fsd *DirectoryStorageDriver) loadFromFile() error {
	data, err := ioutil.ReadFile(path.Join(fsd.FilePath, zoneStaticFile))

	if err != nil {
		return err
	}

	if err = json.Unmarshal(data, &fsd.zoneStaticConfig); err != nil {
		return err
	}

	for _, zone := range fsd.zoneStaticConfig.Zones {
		fsd.zoneMap[zone.Id] = zone
	}

	return nil
}

