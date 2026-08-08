package postgres

import (
	"strings"

	"github.com/jackc/pgx/v5"
)

// rowScanner is implemented by pgx.Row and pgx.Rows, letting scanner helpers
// work with both single-row and multi-row queries.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanMonitorRow adapts a pgx.Row into the rowScanner interface.
func scanMonitorRow(row pgx.Row) rowScanner {
	return row
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
