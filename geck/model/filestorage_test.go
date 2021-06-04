package model

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
	"time"
)

func TestNewDirectoryStorageDriver(t *testing.T) {
	// TODO: replace with fs fixture
	dir := "/tmp/test-x"
	os.MkdirAll(dir, 0777)

	func() {
		file, err := os.Open("test-zones.conf.json")
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
		}
		defer file.Close()

		outFile, err := os.Create(dir + "/zones.conf.json")
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
		}
		defer outFile.Close()

		io.Copy(outFile, file)
	}()

	driver := NewDirectoryStorageDriver(dir)
	var _ StorageDriver = driver

	list, err := driver.doLoadZones()

	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
	}

	require.Nil(t, err)

	for _, info := range list {
		mm, _ := json.Marshal(info)
		os.Stdout.WriteString(string(mm) + "\n")
	}

	list[0].Lane = "0"
	list[1].Lane = "0"
	list[0].ZoneInfoStatic.Name = "test1"

	err = driver.doSaveZone(saveZoneContext{
		zone: &list[0].ZoneInfoStatic,
	})
	require.NoError(t, err)

	list[0].ZoneState.Runtime = 5 * time.Second
	err = driver.doUpdateZoneState(updateZoneStateContext{
		zoneId: list[0].Id,
		state:  &list[0].ZoneState,
	})
	require.NoError(t, err)

	list, _ = driver.doLoadZones()
	require.Equal(t, "test1", list[0].ZoneInfoStatic.Name)

	err = driver.doSaveZone(saveZoneContext{
		zone: &list[0].ZoneInfoStatic,
	})
	require.NoError(t, err)
}