package dynsql

import (
	"database/sql/driver"
)

type JSONType string

const (
	Number  JSONType = "number"
	String  JSONType = "string"
	Boolean JSONType = "boolean"
)

type DYNSQLDriverQuerySet struct {
	GetAllTableNames               string
	GetAllColumnNamesFromTableName string
	GetAllColumnTypesFromTableName string
	AddColumnWithNameFromTableName func(string, JSONType) string
	UpdateColumnTypeFromTableName  func(string, JSONType) string
}

type DYNSQLDriver struct {
	mountedOnDriver driver.Driver
	QuerySet        DYNSQLDriverQuerySet
}

func (d *DYNSQLDriver) toConnector() (driver.Connector, error) {

}

func (d *DYNSQLDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.mountedOnDriver.Open(name)
	if err != nil {
		return nil, err
	}

	tx, err := conn.Begin()

}

func (d *DYNSQLDriver) OpenConnector(name string) (driver.Connector, error) {

}
