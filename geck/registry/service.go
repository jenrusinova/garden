package registry

import (
	"container/list"
	"fmt"
	"log"
	"time"
)

type Service interface {
	/// Start the driver
	Startup() error

	/// Stop the driver
	Shutdown()
}

type ServiceRegistry struct {
	services map[string] Service

	shutdownSeq *list.List
	timeout     time.Duration

	provides map[string] []string
	depends  map[string] []string
}

func (reg *ServiceRegistry) Startup() error {
	waiting := map[string]struct{}{}
	ready   := map[string]struct{}{}
	timeout := time.NewTimer(reg.timeout)
	started := make(chan string, len(reg.services))
	errors  := make(chan error, len(reg.services))

	canStart := func(name string) bool {
		for _, dep := range reg.depends[name] {
			if _, ok := ready[dep]; !ok {
				return false
			}
		}

		return true
	}

	tryStart := func(name string) {
		if !canStart(name) {
			return
		}

		waiting[name] = struct{}{}

		go func() {
			log.Printf("Starting service : " + name)

			if err := reg.services[name].Startup(); err != nil {
				errors <- err
			}

			started <- name
		}()
	}

	for serviceName := range reg.services {
		tryStart(serviceName)
	}

	if len(waiting) == 0 {
		return fmt.Errorf("no services could be started (no services provided or circular dependency)")
	}

	for len(waiting) > 0 {
		select {
		case err := <-errors:
			return err
		case svc := <-started:
			reg.shutdownSeq.PushFront(svc)
			delete(waiting, svc)
			ready[svc] = struct{}{}

			for _, pp := range reg.provides[svc] {
				tryStart(pp)
			}

			if len(waiting) == 0 {
				return nil
			}
		case <-timeout.C:
			panic(fmt.Errorf("timeout while starting services"))
		}
	}

	if reg.shutdownSeq.Len() != len(reg.services) {
		panic(fmt.Errorf("some services could not be started (possible circular dependency?)"))
	}

	return nil
}

func (reg *ServiceRegistry) Shutdown() {
	front := reg.shutdownSeq.Front()

	for front != nil {
		svc := reg.shutdownSeq.Remove(front).(string)
		log.Printf("Shutdown : " + svc)
		reg.services[svc].Shutdown()
		front = reg.shutdownSeq.Front()
	}
}

func NewServiceRegistry() ServiceRegistry {
	return ServiceRegistry{
		services: make(map[string]Service),
		depends:  make(map[string][]string),
		provides: make(map[string][]string),
		shutdownSeq: list.New(),
		timeout: 15 * time.Second,
	}
}

func (reg *ServiceRegistry) AddServiceDep(name string, service Service, dependsOn ...string) {
	reg.services[name] = service
	reg.depends[name] = dependsOn

	for _, dep := range dependsOn {
		reg.provides[dep] = append(reg.provides[dep], name)
	}
}

func (reg *ServiceRegistry) AddService(name string, service Service) {
	reg.services[name] = service
}

var _ Service = &ServiceRegistry{}

func TryRunAsService(serviceTest interface{}) (func(), error) {
	if service, ok := serviceTest.(Service); ok {
		if err := service.Startup(); err != nil {
			return nil, err
		}

		return service.Shutdown, nil
	}

	return nil, nil
}
