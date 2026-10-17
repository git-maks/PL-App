// Package migrations embeds the SQL schema migrations, applied in file-name order.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
