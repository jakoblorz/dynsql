package dynsql

import (
	"database/sql/driver"
	"errors"
	"sync"
)

var (
	ErrFailedToParseString = errors.New("failed to parse string")
)

type DriverQueryDefinition struct {
	GetAllTableNames               string
	GetAllColumnsFromTableName     string
	AddColumnWithNameFromTableName string

	InsertValuesOnTableNameWithColumnNames func([]string) string
}

type Driver struct {
	baseDriver driver.Driver
	queries    DriverQueryDefinition

	schemas map[string][]string
	mu      sync.Locker
}

func WrapDriver(baseDriver driver.Driver, queries DriverQueryDefinition) *Driver {
	return &Driver{
		baseDriver: baseDriver,
		queries:    queries,
	}
}

func (d *Driver) Open(name string) (driver.Conn, error) {
	conn, err := d.baseDriver.Open(name)
	if err != nil {
		return nil, err
	}
	return newConn(d, conn)
}
