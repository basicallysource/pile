package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/basicallysource/pile/internal/catalog"
)

// buildPicture serves a build's picture from the data folder, or sends the
// browser to its set's until one is drawn (cmd/render-builds), so new
// pictures show without a restart.
func (s *Service) buildPicture(w http.ResponseWriter, r *http.Request) {
	num := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, catalog.BuildPicturesURL), ".png")
	b := s.cat.Sets[num]
	if b == nil || b.Kind != catalog.Build {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(catalog.BuildPictures(s.dataDir), catalog.BuildPicture(b.Num))
	if _, err := os.Stat(path); err != nil {
		if b.Of.ImageURL == "" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, b.Of.ImageURL, http.StatusFound)
		return
	}
	http.ServeFile(w, r, path)
}
