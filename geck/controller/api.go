package controller

import (
	"encoding/json"
	"geck/model"
	"geck/web"
	"log"
	"net/http"
	"regexp"
	"time"
)


var defaultServerAddr = "0.0.0.0:8089"


type ZoneAction struct {
	ZoneId   string `json:"zone_id"`
	Version	 uint64 `json:"version"`
	Action   string `json:"action"`
	Duration time.Duration `json:"for"`
}


type Response struct {
	Status string            `json:"status"`
	Error  string            `json:"error,omitempty"`
	Zone   []*model.ZoneInfo `json:"zones"`
}

type GardenAPI struct {
	*web.HttpService
	webData    web.Directory
	controller *GardenController
}

func NewGardenAPI(controller *GardenController, webData web.Directory) *GardenAPI {
	result := &GardenAPI{
		HttpService: web.NewHttpServer(defaultServerAddr),
		webData:     webData,
		controller:  controller,
	}

	log.Printf("Listening at %s", defaultServerAddr)

	result.RegisterDirectory(webData, "/")
	result.RegisterStartupHandler(result.PrepareHttp)

	return result
}

func (api * GardenAPI) HandleZoneInfo(writer http.ResponseWriter, req *http.Request) {
	log.Printf("Http request: %s, from : %s", req.URL.Path, req.RemoteAddr)

	zones := api.controller.GetZoneInfo("")

	writer.Header().Add("Content-Type", "application/json")

	data, err := json.Marshal(Response{
		Status: "OK",
		Zone:   zones,
	})

	if err != nil {
		log.Printf("Http error : %s", err.Error())
		http.Error(writer, err.Error(), 503)
		return
	}

	_, err = writer.Write(data)

	if err != nil {
		log.Printf("Http error : %s", err.Error())
	}
}

func (api * GardenAPI) HandleZoneUpdate(writer http.ResponseWriter, req *http.Request) {
}

type APIContext struct {
	Writer http.ResponseWriter
	Request *http.Request
	PathParts []string
}

func WrapAPICall(
	handler func (context APIContext) error,
	re * regexp.Regexp) func
	(writer http.ResponseWriter, req *http.Request) {
	return func(writer http.ResponseWriter, req *http.Request) {
		log.Printf("Http request: %s, from : %s", req.URL.Path, req.RemoteAddr)

		err := handler(APIContext{
			Writer:    writer,
			Request:   req,
			PathParts: re.FindStringSubmatch(req.URL.Path),
		})

		if err != nil {
			log.Printf("Controller error : %s", err.Error())
			http.Error(writer, err.Error(), 503)
		}
	}
}

func (api * GardenAPI) HandleZoneStop(context APIContext) error {
	log.Printf("Http, stop zone req: %s", context.PathParts[1])

	if err := api.controller.StopZone(context.PathParts[1]); err != nil {
		return err
	}

	return nil
}

func (api * GardenAPI) HandleZoneStart(context APIContext) error {
	log.Printf("Http, start zone req: %s", context.PathParts[1])

	if err := api.controller.StartZone(context.PathParts[1], 5 * time.Minute, true); err != nil {
		return err
	}

	return nil
}

func (api * GardenAPI) PrepareHttp() error {
	api.Mux().HandleFunc("/zone/", api.HandleZoneInfo)

	api.Mux().HandleFunc("/start/",
		WrapAPICall(
			api.HandleZoneStart,
			regexp.MustCompile("/start/([a-zA-Z0-9\\-]+)")))

	api.Mux().HandleFunc("/stop/",
		WrapAPICall(
			api.HandleZoneStop,
			regexp.MustCompile("/stop/([a-zA-Z0-9\\-]+)")))

	return nil
}

