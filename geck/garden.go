package main

import (
	"flag"
	"geck/controller"
	"geck/driver"
	"geck/model"
	"geck/registry"
	"geck/web"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetFlags(log.Ldate|log.Ltime|log.Lmicroseconds)
	log.SetOutput(os.Stderr)

	ioDriver, err := driver.CreateIODriver()

	if err != nil {
		log.Printf("error : %s", err.Error())
	}

	var dataDirectory string
	var webDataFile string

	flag.StringVar(&dataDirectory, "data", "./data",
		"Directory with configuration and run files")

	flag.StringVar(&webDataFile, "web-data", "./garden-webdata.tar.gz",
		"Tar file with web data")

	flag.Parse()

	services := registry.NewServiceRegistry()
	storage := model.NewDirectoryStorageDriver(dataDirectory)
	gc := controller.NewGardenController(ioDriver, storage)
	webData := web.NewTarMap(webDataFile, "/var/tmp/geck/web")
	api := controller.NewGardenAPI(gc, webData)

	services.AddService("storage", storage)
	services.AddServiceDep("controller", gc, "storage", "io_driver")
	services.AddService("web_data", webData)
	services.AddService("io_driver", ioDriver.(registry.Service))
	services.AddServiceDep("http_server", api, "web_data", "controller")

	if err := services.Startup(); err != nil {
		log.Fatalf("Startup error : %s", err.Error())
	}

	defer services.Shutdown()

	signalHandler := make(chan os.Signal, 8)
	signal.Notify(signalHandler, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-signalHandler:
			return
		}
	}
}
