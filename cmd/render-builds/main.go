// render-builds draws a picture of every build within a set (from the 3D
// models of sets in the data folder's set-models/) into its pictures/
// folder, which pile serves; a build has its set's picture until it has its
// own. It needs the LDraw parts library in the data folder's ldraw/. A
// build drawn before is drawn again only with -again.
package main

import (
	"flag"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/ldraw"
)

func main() {
	data := flag.String("data", "data", "pile's data folder")
	size := flag.Int("size", 320, "the pictures' width and height, in pixels")
	again := flag.Bool("again", false, "draw builds drawn before again")
	only := flag.String("set", "", "draw only this set's builds")
	flag.Parse()
	cat, err := catalog.Load(*data)
	if err != nil {
		log.Fatal(err)
	}
	d := ldraw.NewDrawer(ldraw.OpenLibrary(filepath.Join(*data, "ldraw")))
	out := catalog.BuildPictures(*data)
	var builds []*catalog.Set
	for _, s := range cat.Sets {
		if s.Kind == catalog.Build && (*only == "" || s.Of.Num == *only) {
			builds = append(builds, s)
		}
	}
	sort.Slice(builds, func(i, j int) bool { return builds[i].Num < builds[j].Num })
	var done, skipped atomic.Int32
	work := make(chan *catalog.Set)
	var wg sync.WaitGroup
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for b := range work {
				path := filepath.Join(out, catalog.BuildPicture(b.Num))
				if _, err := os.Stat(path); err == nil && !*again {
					skipped.Add(1)
					continue
				}
				if err := write(path, d, b, *size); err != nil {
					log.Printf("%s: %v", b.Num, err)
					continue
				}
				if n := done.Add(1); n%100 == 0 {
					log.Printf("%d drawn", n)
				}
			}
		}()
	}
	for _, b := range builds {
		work <- b
	}
	close(work)
	wg.Wait()
	log.Printf("%d builds drawn, %d drawn before, into %s", done.Load(), skipped.Load(), out)
}

// write draws b into path, through a file beside it, so a picture is never
// served half written.
func write(path string, d *ldraw.Drawer, b *catalog.Set, size int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path + ".new")
	if err != nil {
		return err
	}
	if err := png.Encode(f, d.Draw(b.Parts, size)); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(path+".new", path)
}
