// Package content embeds the knowledge-base JSON files so the Go backend uses exactly the same
// archetypes, programmes, clubs and demo event as the web app.
package content

import "embed"

//go:embed *.json
var FS embed.FS
