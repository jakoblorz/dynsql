package sqlite3

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/jakoblorz/dynsql"
	"github.com/jakoblorz/dynsql/x/util"
)

const SelectAllTableNamesSQL = `SELECT name FROM sqlite_master 
	WHERE type = 'table';`

const SelectAllTableColumnsSQL = `SELECT sql FROM sqlite_master
	WHERE type = 'table' AND name = '%s';`

const CreateNewTableSQL = `CREATE TABLE %s (
	%s
);`

const AlterTableAddColumnSQL = `ALTER TABLE %s
	ADD COLUMN $1 $2;`

const InsertValuesSQL = `INSERT INTO %s (
	%s
) VALUES (
	%s
);`

const DefaultFallbackType = "TEXT"

var (
	DefaultTypeMap = map[string]string{
		dynsql.InheritedIDField:         fmt.Sprintf("%s PRIMARY KEY", DefaultFallbackType),
		dynsql.InheritedUpdatedAtField:  DefaultFallbackType,
		dynsql.InheritedInsertedAtField: DefaultFallbackType,
	}
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
		if len(row) >= 1 {
			names = append(names, row[0].(string))
		}
	}
	err = <-errCh
	if err != nil && err != io.EOF {
		sqlErr, ok := err.(interface{ Error() string })
		if ok && sqlErr.Error() == "not an error" {
			return nil, sql.ErrNoRows
		}

		return nil, err
	}
	return names, nil
}

func (s SQLite3Dialect) GetAllTableColumns(tableName string, x dynsql.ExecerQueryer) (map[string]string, error) {
	rows, err := x.Query(fmt.Sprintf(SelectAllTableColumnsSQL, tableName), nil)
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
		if len(row) >= 1 {
			stmts = append(stmts, row[0].(string))
		}
	}
	err = <-errCh
	if err != nil && err != io.EOF {
		sqlErr, ok := err.(interface{ Error() string })
		if ok && sqlErr.Error() == "not an error" {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	v := map[string]string{}
	if len(stmts) == 0 {
		return v, nil
	}

	sql := stmts[0]
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

func (s SQLite3Dialect) CreateNewTable(tableName string, keys []string, x dynsql.ExecerQueryer) error {
	query := ""
	for _, k := range keys {
		t, ok := DefaultTypeMap[k]
		if !ok {
			t = DefaultFallbackType
		}
		query = fmt.Sprintf("%s, %s %s", query, k, t)
	}
	query = query[2:]

	_, err := x.Exec(fmt.Sprintf(CreateNewTableSQL, tableName, query), nil)
	if err != nil {
		sqlErr, ok := err.(interface{ Error() string })
		if ok && sqlErr.Error() == "not an error" {
			return nil
		}
	}
	return err
}

func (s SQLite3Dialect) AddColumnToTable(tableName, key string, x dynsql.ExecerQueryer) error {
	t, ok := DefaultTypeMap[key]
	if !ok {
		t = DefaultFallbackType
	}

	_, err := x.Exec(fmt.Sprintf(AlterTableAddColumnSQL, tableName), []driver.Value{key, t})
	if err != nil {
		sqlErr, ok := err.(interface{ Error() string })
		if ok && sqlErr.Error() == "not an error" {
			return nil
		}
	}
	return err
}

func (s SQLite3Dialect) InsertValuesPrepare(tableName string, keys []string, x dynsql.ExecerQueryerPreparer) (driver.Stmt, error) {
	columnNames := ""
	placeholders := ""
	for i, k := range keys {
		columnNames = fmt.Sprintf("%s, %s", columnNames, k)
		placeholders = fmt.Sprintf("%s, $%d", placeholders, i+1)
	}
	columnNames = columnNames[2:]
	placeholders = placeholders[2:]

	return x.Prepare(fmt.Sprintf(InsertValuesSQL, tableName, columnNames, placeholders))
}
