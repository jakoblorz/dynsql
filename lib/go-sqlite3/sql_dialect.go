package sqlite3

import (
	"database/sql/driver"
	"regexp"
	"strings"

	"github.com/jakoblorz/dynsql"
	"github.com/jakoblorz/dynsql/x/util"
)

const SelectAllTableNamesSQL = `SELECT name FROM sqlite_master 
	WHERE type = "table";`

const SelectAllTableColumnsSQL = `SELECT sql FROM sqlite_master
	WHERE type = "table" AND name = $1;`

var (
	ColumnNameAndTypeRegex = regexp.MustCompile(`(?:\(|\,\s*)(?P<Key>[a-zA-Z\_\-]+)(?:\s*)(?P<Type>[a-zA-Z\_\-]+)(?:\s*)`)
)

type SQLite3Dialect int

func (s SQLite3Dialect) GetAllTableNames(x dynsql.ExecerQueryer) ([]string, error) {
	rows, err := x.Query(SelectAllTableNamesSQL, nil)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	names := []string{}
	vCh, errCh := util.IterateDriverRows(rows)
	for {
		row, ok := <-vCh
		if !ok {
			break
		}
		names = append(names, row[0].(string))
	}
	err = <-errCh
	if err != nil {
		return nil, err
	}
	return names, nil
}

func (s SQLite3Dialect) GetAllTableColumns(tableName string, x dynsql.ExecerQueryer) (map[string]string, error) {
	rows, err := x.Query(SelectAllTableColumnsSQL, []driver.Value{tableName})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stmts := []string{}
	vCh, errCh := util.IterateDriverRows(rows)
	for {
		row, ok := <-vCh
		if !ok {
			break
		}
		stmts = append(stmts, row[0].(string))
	}
	err = <-errCh
	if err != nil {
		return nil, err
	}

	sql := stmts[0]
	v := map[string]string{}
	m := ColumnNameAndTypeRegex.FindStringSubmatch(sql)
	for ; len(m) != 0; m = ColumnNameAndTypeRegex.FindStringSubmatch(sql) {
		sql = strings.Replace(sql, m[0], "", -1)
		key := ""
		typ := ""
		for i, k := range ColumnNameAndTypeRegex.SubexpNames() {
			if k == "Key" {
				key = m[i]
			}
			if k == "Type" {
				typ = m[i]
			}
		}
		v[key] = typ
	}

	return v, nil
}

func (s SQLite3Dialect) CreateNewTable(table string, keys []string, x dynsql.ExecerQueryer) error {
	return nil

}

func (s SQLite3Dialect) AddColumnToTable(table, key string, x dynsql.ExecerQueryer) error {
	return nil
}

func (s SQLite3Dialect) InsertValuesPrepare(table string, keys []string, x dynsql.ExecerQueryerPreparer) (driver.Stmt, error) {
	return nil, nil
}
