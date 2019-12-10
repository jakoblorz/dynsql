package dynsql

import (
	"database/sql/driver"
	"errors"
)

var (
	ErrFailedToParseString = errors.New("failed to parse string")
)

type AbstractColumnType string

const (
	Number  AbstractColumnType = "number"
	String  AbstractColumnType = "string"
	Boolean AbstractColumnType = "boolean"
)

type DriverQuerySet struct {
	GetAllTableNames               string
	GetAllColumnsFromTableName     string
	AddColumnWithNameFromTableName func(string, AbstractColumnType) string
	UpdateColumnTypeFromTableName  func(string, AbstractColumnType) string
	
	UpsertValuesOnTableNameWithColumnNames func(string[]) string

	TranslateSQLType func(string) (AbstractColumnType, error)
}

type Driver struct {
	mountedOnDriver driver.Driver
	driverQuerySet  DriverQuerySet

	polledTableSchemas map[string][]*ColumnDefinition
}

type ColumnDefinition struct {
	Name string
	Type AbstractColumnType
}

func CreateDriver(mountedOnDriver driver.Driver, driverQuerySet DriverQuerySet) *Driver {
	return &Driver{
		mountedOnDriver: mountedOnDriver,
		driverQuerySet:  driverQuerySet,
	}
}

func (d *Driver) pollDatabaseData(conn driver.Conn) error {
	getAllTableNamesStmt, err := conn.Prepare(d.driverQuerySet.GetAllTableNames)
	if err != nil {
		return err
	}

	getAllTableNamesRows, err := getAllTableNamesStmt.Query(nil)
	if err != nil {
		return err
	}
	defer getAllTableNamesRows.Close()

	iterRowSlice, polledTableSchemas := []driver.Value{}, map[string][]*ColumnDefinition{}
	for err = getAllTableNamesRows.Next(iterRowSlice); err == nil; err = getAllTableNamesRows.Next(iterRowSlice) {
		if name, ok := iterRowSlice[0].(string); ok {
			polledTableSchemas[name] = make([]*ColumnDefinition, 0)
			continue
		}

		return ErrFailedToParseString
	}
	for table := range polledTableSchemas {
		polledTableSchemas[table], err = d.pollColumnDefinitions(conn, table)
		if err != nil {
			return err
		}
	}
	d.polledTableSchemas = polledTableSchemas

	return nil
}

func (d *Driver) pollColumnDefinitions(conn driver.Conn, tableName string) ([]*ColumnDefinition, error) {
	getAllColumnsFromTableNameStmt, err := conn.Prepare(d.driverQuerySet.GetAllColumnsFromTableName)
	if err != nil {
		return nil, err
	}
	defer getAllColumnsFromTableNameStmt.Close()

	getAllColumnsFromTableNameRows, err := getAllColumnsFromTableNameStmt.Query([]driver.Value{tableName})
	if err != nil {
		return nil, err
	}

	iterRowSlice, columnDefinitions := []driver.Value{}, []*ColumnDefinition{}
	for err = getAllColumnsFromTableNameRows.Next(iterRowSlice); err == nil; err = getAllColumnsFromTableNameRows.Next(iterRowSlice) {
		var err error

		var atype AbstractColumnType
		var stype string

		sname, ok := iterRowSlice[0].(string)
	HANDLE_ERROR:
		if ok {
			stype, ok = iterRowSlice[1].(string)
			if !ok {
				goto HANDLE_ERROR
			}

			atype, err = d.driverQuerySet.TranslateSQLType(stype)
			if err != nil {
				ok = false
				goto HANDLE_ERROR
			}

			columnDefinitions = append(columnDefinitions, &ColumnDefinition{
				Name: sname,
				Type: atype,
			})
		}

		if !ok {
			if err != nil {
				return nil, err
			}

			return nil, ErrFailedToParseString
		}
	}

	return columnDefinitions, nil
}

func (d *Driver) Open(name string) (driver.Conn, error) {
	conn, err := d.mountedOnDriver.Open(name)
	if err != nil {
		return nil, err
	}

	err = d.pollDatabaseData(conn)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
