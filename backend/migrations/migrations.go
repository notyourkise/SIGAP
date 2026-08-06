// Package migrations menyematkan (embed) seluruh file migration SQL versioned
// golang-migrate v4 ke dalam binary aplikasi. Package ini dikonsumsi oleh
// config.RunMigrations (saat startup) dan CLI cmd/migrate/main.go (manual)
// sehingga skema database selalu dikelola dari satu sumber: folder migrations/.
package migrations

import "embed"

// FS berisi file migration yang di-compile ke dalam binary
// (format: <urutan>_<nama>.up.sql / <nama>.down.sql).
//
//go:embed *.sql
var FS embed.FS