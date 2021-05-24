package driver

import (
	"bufio"
	"fmt"
	"github.com/stianeikeland/go-rpio"
	"log"
	"os"
	"runtime"
	"strings"
)

/// RPIOPin pin implementation
type RPIOPin struct {
	Pin  rpio.Pin
	id   string
	isOn bool
}

/// GetID get pin name
func (rpp * RPIOPin) GetID() string {
	return rpp.id
}

/// IsRunning get pin running
func (rpp * RPIOPin) IsRunning() bool {
	return rpp.isOn
}

/// Start - activate the pin
func (rpp * RPIOPin) Start() {
	rpp.Pin.Low()
	rpp.isOn = true
}

/// Stop - deactivate the pin
func (rpp * RPIOPin) Stop() {
	rpp.Pin.High()
	rpp.isOn = false
}

/// Raspberry Pi Pin driver based on rpio
type RaspberryDriver struct {
	pinMap  map[string]WireActor
	version int
}

/// AvailableActors enumerate available pins
func (rpiod *RaspberryDriver) AvailableActors() []WireActor {
	result := make([]WireActor, len(rpiod.pinMap))
	i := 0

	for _, actor := range rpiod.pinMap {
		result[i] = actor
		i++
	}

	return result
}

const (
	UNKNOWN = 0
	DRIVER_PI1_A = 11
	DRIVER_PI1_B = 12
)

func parseModelLine(text string) (int, bool) {
	if !strings.HasPrefix(strings.ToLower(text), "model") {
		return 0, false
	}

	i := strings.Index(text, ": ")

	if i > 0 {
		text = text[i+2:]
	}

	log.Printf("Hardware model : %s", text)

	if strings.Contains(text, "Raspberry Pi Model B") {
		if strings.Contains(text, "Rev 2") {
			return DRIVER_PI1_B, true
		}

		return DRIVER_PI1_A, true
	}

	return 0, false
}

func CreateIODriver() (WireDriver, error) {
	if runtime.GOARCH == "amd64" {
		return testDriver, nil
	}

	cpuinfo, err := os.Open("/proc/cpuinfo")

	if err != nil {
		panic(fmt.Errorf("unable to determine Hardware version : %s", err.Error()))
	}

	defer cpuinfo.Close()

	scanner := bufio.NewScanner(cpuinfo)

	var driver = UNKNOWN

	for scanner.Scan() {
		var ok bool

		if driver, ok = parseModelLine(scanner.Text()); ok {
			break
		}
	}

	if driver == UNKNOWN {
		log.Print("WARNING! Unable to find driver, using console noop driver!")
		return testDriver, nil
	}

	return &RaspberryDriver{
		pinMap: map[string]WireActor{},
		version: driver,
	}, nil
}

/// Startup the driver
func (rpiod *RaspberryDriver) Startup() error {
	if err := rpio.Open(); err != nil {
		return err
	}

	createPin := func(id string, pin int) {
		iopin := &RPIOPin{
			id:   id,
			Pin:  rpio.Pin(pin),
			isOn: false,
		}

		rpiod.pinMap[id] = iopin
		iopin.Pin.Output()
		iopin.Pin.High()
	}

	if rpiod.version == DRIVER_PI1_B || rpiod.version == DRIVER_PI1_A {
		createPin("gpio7", 4)
		createPin("gpio0", 17)
		createPin("gpio1", 18)

		if rpiod.version == DRIVER_PI1_A {
			createPin("gpio2", 21)
		} else {
			createPin("gpio2", 27)
		}

		createPin("gpio3", 22)
		createPin("gpio4", 23)
		createPin("gpio5", 24)
		createPin("gpio6", 25)
	}

	return nil
}


/// Shutdown driver
func (rpiod *RaspberryDriver) Shutdown() {
	for _, actor := range rpiod.pinMap {
		actor.Stop()
	}

	_ = rpio.Close()
}

