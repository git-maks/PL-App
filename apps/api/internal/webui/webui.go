// Package webui embeds the built web app (apps/web/build, copied to ./dist by `make build`
// or the Dockerfile). When dist is empty the server runs API-only.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the built web app, or nil when it was not embedded.
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil
	}
	return sub
}
