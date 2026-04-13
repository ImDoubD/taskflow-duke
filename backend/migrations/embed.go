// Package migrations embeds all SQL migration files into the binary.
// golang-migrate reads them at runtime via the iofs source driver.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
