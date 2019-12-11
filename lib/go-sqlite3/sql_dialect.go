package sqlite3

import (
	"database/sql/driver"

	"github.com/jakoblorz/dynsql"
)

type SQLiteDialect int

func (s SQLiteDialect) GetAllTableNames(x dynsql.ExecerQueryer) ([]string, error) {
	return nil, nil
}

func (s SQLiteDialect) GetAllTableColumns(tableName string, x dynsql.ExecerQueryer) (map[string]string, error) {
	return nil, nil
}

func (s SQLiteDialect) CreateNewTable(table string, keys []string, x dynsql.ExecerQueryer) error {
	return nil

}

func (s SQLiteDialect) AddColumnToTable(table, key string, x dynsql.ExecerQueryer) error {
	return nil
}

func (s SQLiteDialect) InsertValuesPrepare(table string, keys []string, x dynsql.ExecerQueryerPreparer) (driver.Stmt, error) {
	return nil, nil
}
