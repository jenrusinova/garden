package web

import (
	"container/list"
	"geck/registry"
	"log"
	"net"
	"net/http"
	"time"
)

type Entry interface {
	GetName() string
	GetData() ([]byte, error)
}

type Directory interface {
	GetEntries() map[string]Entry
	GetWebHandlerFor(name string, checkUri string) func(writer http.ResponseWriter, req *http.Request)
	SetWebHandlers(context string, mux *http.ServeMux, defaultName string)
}

type StartupHandler func() error

type HttpService struct {
	http.Server
	startupHandlers list.List
}

func (t *HttpService) Mux() *http.ServeMux {
	return t.Handler.(*http.ServeMux)
}

func (t *HttpService) RegisterStartupHandler(handler StartupHandler) {
	t.startupHandlers.PushBack(handler)
}

func (t *HttpService) RegisterDirectory(dir Directory, context string) {
	t.RegisterStartupHandler(func() error {
		dir.SetWebHandlers(context, t.Mux(), "index.html")
		return nil
	})
}

func NewHttpServer(addr string) *HttpService {
	result := &HttpService{}
	result.Init(addr)
	return result
}

func (t *HttpService) Init(addr string) {
	t.Server = http.Server {
		Addr:              addr,
		Handler:           http.NewServeMux(),
		ReadTimeout:       20 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      20 * time.Second,
	}
}

func (t *HttpService) Startup() error {
	ln, err := net.Listen("tcp", t.Addr)

	if err != nil {
		return err
	}

	for handler := t.startupHandlers.Front(); handler != nil; handler = handler.Next() {
		if err := handler.Value.(StartupHandler)(); err != nil {
			return err
		}
	}

	go func() {
		if err := t.Serve(ln); err != http.ErrServerClosed && err != nil {
			log.Fatalf("Web server down : %s", err.Error())
		}
	}()

	return nil
}

func (t *HttpService) Shutdown() {
	_ = t.Close()
}

var _ registry.Service = &HttpService{}

