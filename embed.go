// Package pile holds the app's static build, embedded into the server binary.
package pile

import (
	"embed"
	"io/fs"
)

//go:embed all:app/build
var app embed.FS

// App is the app's static build (app/build).
func App() fs.FS {
	sub, err := fs.Sub(app, "app/build")
	if err != nil {
		panic(err)
	}
	return sub
}
