// Package migrations embeds the SQL schema migrations applied by `idp migrate`.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
