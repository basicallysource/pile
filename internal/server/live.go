package server

import (
	"io/fs"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/basicallysource/pile/gen/pile/v1/pilev1connect"
	"github.com/basicallysource/pile/internal/catalog"
)

// Live is the Service being served. Fresh lots make a new Service off to the
// side, which Swap puts in place whole: the page never goes down, and a
// request is answered by one Service from start to end.
type Live struct{ now atomic.Pointer[Service] }

// NewLive serves s.
func NewLive(s *Service) *Live {
	l := &Live{}
	l.now.Store(s)
	return l
}

// Service is the Service being served.
func (l *Live) Service() *Service { return l.now.Load() }

// Swap serves next from now on and closes the Service it replaces.
func (l *Live) Swap(next *Service) { l.now.Swap(next).Close() }

// Handler serves the API and the pictures of builds (pictures.go) from the
// Service live serves at each request, and the app's static build for every
// other path (index.html for the app's own routes).
func Handler(live *Live, app fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/"+pilev1connect.PileServiceName+"/", func(w http.ResponseWriter, r *http.Request) {
		live.Service().api.ServeHTTP(w, r)
	})
	mux.HandleFunc(catalog.BuildPicturesURL, func(w http.ResponseWriter, r *http.Request) {
		live.Service().buildPicture(w, r)
	})
	files := http.FileServerFS(app)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name != "" {
			if _, err := fs.Stat(app, name); err != nil {
				r.URL.Path = "/"
			}
		}
		files.ServeHTTP(w, r)
	})
	return mux
}
