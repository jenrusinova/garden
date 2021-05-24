package driver

import (
	"fmt"
	"geck/registry"
	"log"
)

type TestActor struct {
	id      string
	started bool
}

func (t *TestActor) GetID() string {
	return t.id
}

func (t *TestActor) IsRunning() bool {
	return t.started
}

func (t *TestActor) Start() {
	fmt.Printf("Started %s\n", t.id)
	t.started = true
}

func (t *TestActor) Stop() {
	fmt.Printf("Stopped %s\n", t.id)
	t.started = false
}

type TestDriver struct {
	actors []*TestActor
}

func (td * TestDriver) Startup() error {
	log.Print("Starting test driver")
	return nil
}

func (td * TestDriver) Shutdown() {
	log.Print("Shutting down test driver\n")
}

func (td * TestDriver) AvailableActors() []WireActor {
	result := make([]WireActor, len(td.actors))

	for i, actor := range td.actors {
		result[i] = actor
	}

	return result
}

var testDriver = &TestDriver{
	actors: []*TestActor{
		{ id: "gpio7" },
		{ id: "gpio0" },
		{ id: "gpio1" },
		{ id: "gpio2" },
		{ id: "gpio3" },
		{ id: "gpio4" },
		{ id: "gpio5" },
		{ id: "gpio6" },
	},
}

var _ WireDriver = testDriver
var _ registry.Service = testDriver
