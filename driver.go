package dynsql

import (
	"database/sql/driver"
	"errors"
	"sync"
)

var (
	ErrFailedToParseString = errors.New("failed to parse string")
)

type Driver struct {
	SQL SQLDialect

	baseDriver driver.Driver
	schemas    map[string]map[string]string
	mu         *sync.RWMutex
}

func WrapDriver(baseDriver driver.Driver, sql SQLDialect) *Driver {
	return &Driver{
		SQL: sql,

		baseDriver: baseDriver,
		mu:         &sync.RWMutex{},
	}
}

func (d *Driver) Open(name string) (driver.Conn, error) {
	conn, err := d.baseDriver.Open(name)
	if err != nil {
		return nil, err
	}
	return newConn(d, conn)
}
