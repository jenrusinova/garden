package web

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FileDesc struct {
	Name     string
	MimeType string
	Size     int

	absFileName string
}

func (f FileDesc) GetName() string {
	return f.Name
}

func (f FileDesc) GetData() ([]byte, error) {
	return ioutil.ReadFile(f.absFileName)
}

type TarMap struct {
	Files map[string] FileDesc
	DataBaseDir string
	SourceFile string
}

func (t *TarMap) GetEntries() map[string]Entry {
	result := map[string]Entry{}

	for k, v := range t.Files {
		result[k] = &v
	}

	return result
}

func (t *TarMap) SetWebHandlers(context string, mux *http.ServeMux, defaultName string) {
	if !strings.HasSuffix(context, "/") {
		context = context + "/"
	}

	for name := range t.Files {
		mux.HandleFunc(context + name, t.GetWebHandlerFor(name, context + name))
	}

	if defaultName != "" {
		mux.HandleFunc(context, t.GetWebHandlerFor(defaultName, context))
	}
}

func writeAll(filename string, reader io.Reader) (int, error) {
	var tmpFile *os.File
	var err error
	var size int64

	if tmpFile, err = os.Create(filename); err != nil {
		return 0, err
	}

	defer tmpFile.Close()

	if size, err = io.Copy(tmpFile, reader); err != nil {
		return 0, err
	}

	return int(size), nil
}

func guessMimeType(fileName string) string {
	switch {
	case strings.HasSuffix(fileName, ".css"):
		return "text/css"
	case strings.HasSuffix(fileName, ".js"):
		return "text/javascript"
	case strings.HasSuffix(fileName, ".html"):
		return "text/html"
	}

	return "text/plain"
}

func (t * TarMap) Update() error {
	var err error
	var osFile *os.File
	var tarFile io.Reader

	if err = os.MkdirAll(t.DataBaseDir, 0700); err != nil {
		return err
	}

	if osFile, err = os.Open(t.SourceFile); err != nil {
		return err
	}
	defer osFile.Close()

	tarFile = osFile

	if strings.HasSuffix(osFile.Name(), ".gz") {
		if tarFile, err = gzip.NewReader(tarFile); err != nil {
			return err
		}
	}

	var tarReader = tar.NewReader(tarFile)

	for {
		var hdr, err = tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if !hdr.FileInfo().Mode().IsRegular() {
			log.Printf("Ignoring file : %s", hdr.FileInfo().Name())
		} else {
			var fileDesc = FileDesc{
				Name: hdr.FileInfo().Name(),
				absFileName: filepath.Join(t.DataBaseDir, hdr.FileInfo().Name()),
			}

			if fileDesc.Size, err = writeAll(fileDesc.absFileName, tarReader); err != nil {
				return err
			}

			fileDesc.MimeType = guessMimeType(fileDesc.absFileName)
			t.Files[fileDesc.Name] = fileDesc

			log.Printf("Found file : %s, putting to : %s", fileDesc.Name, fileDesc.absFileName)
		}
	}

	return nil
}

func (t * TarMap) Startup() error {
	return t.Update()
}

func (t * TarMap) Shutdown() {
	_ = os.RemoveAll(t.DataBaseDir)
}

func NewTarMap(tarFile string, baseTempDir string) *TarMap {
	return &TarMap{
		Files: make(map[string]FileDesc),
		DataBaseDir: baseTempDir,
		SourceFile:  tarFile,
	}
}

func (t * TarMap) GetWebHandlerFor(name string, checkUri string) func(writer http.ResponseWriter, req *http.Request) {
	var fDsc FileDesc
	var ok bool

	if fDsc, ok = t.Files[name]; !ok {
		panic("entry not found")
	}

	return func(writer http.ResponseWriter, req *http.Request) {
		if req.RequestURI != checkUri {
			log.Printf("Http request to: %s, ignoring", req.RequestURI)
			writer.WriteHeader(404)
			_, _ = writer.Write([]byte("Not found"))
			return
		}

		log.Printf("Http request: %s, from : %s, handled by file %s",
			req.URL.Path, req.RemoteAddr, fDsc.absFileName)

		data, err := ioutil.ReadFile(fDsc.absFileName)

		if err != nil {
			log.Printf("Web error : %s", err.Error())
			writer.WriteHeader(404)
			_, _ = writer.Write([]byte("Internal error: cannot read file"))
			return
		}

		writer.Header().Add("Content-Type", fDsc.MimeType)
		writer.Header().Add("Cache-Control", "public, max-age=3600")
		writer.WriteHeader(200)
		_, _ = writer.Write(data)
	}
}


var _ Directory = &TarMap{}
var _ Entry = &FileDesc{}


