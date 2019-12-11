package sqlite3

import (
	"database/sql"

	"github.com/jakoblorz/dynsql"
	sqlite3 "github.com/mattn/go-sqlite3"
)

func init() {
	sql.Register("dyn-sqlite3", dynsql.WrapDriver(&sqlite3.SQLiteDriver{}, SQLiteDialect(0)))
}
