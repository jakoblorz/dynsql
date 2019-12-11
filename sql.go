package dynsql

import "database/sql/driver"

type ExecerQueryer interface {
	driver.Queryer
	driver.Execer
}

type ExecerQueryerPreparer interface {
	ExecerQueryer
	Prepare(query string) (driver.Stmt, error)
}

type SQLDialect interface {
	GetAllTableNames(x ExecerQueryer) ([]string, error)
	GetAllTableColumns(tableName string, x ExecerQueryer) (map[string]string, error)

	CreateNewTable(table string, keys []string, x ExecerQueryer) error
	AddColumnToTable(table, key string, x ExecerQueryer) error

	InsertValuesPrepare(table string, keys []string, x ExecerQueryerPreparer) (driver.Stmt, error)
}
